// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"golang.org/x/text/language"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
)

const (
	xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

const streamIDLength = 16

// SessionState represents the current state of an XMPP session. For a
// description of each bit, see the various SessionState typed constants.
type SessionState uint8

const (
	// Secure indicates that the underlying connection has been secured. For
	// instance, after STARTTLS has been performed or if a pre-secured connection
	// is being used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Authn indicates that the session has been authenticated (probably with
	// SASL).
	Authn

	// Ready indicates that the session is fully negotiated and that XMPP stanzas
	// may be sent and received.
	Ready

	// Received indicates that the session was initiated by a foreign entity.
	Received

	// OutputStreamClosed indicates that the output stream has been closed with a
	// stream end tag.  When set all write operations will return an error even if
	// the underlying TCP connection is still open.
	OutputStreamClosed

	// InputStreamClosed indicates that the input stream has been closed with a
	// stream end tag. When set all read operations will return an error.
	InputStreamClosed
)

type stream struct {
	to      *jid.JID
	from    *jid.JID
	id      string
	version internal.Version
	xmlns   string
	lang    language.Tag
}

// This MUST only return stream errors.
func streamFromStartElement(s xml.StartElement) (stream, error) {
	stream := stream{}
	for _, attr := range s.Attr {
		switch attr.Name {
		case xml.Name{Space: "", Local: "to"}:
			stream.to = &jid.JID{}
			if err := stream.to.UnmarshalXMLAttr(attr); err != nil {
				return stream, streamerror.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			stream.from = &jid.JID{}
			if err := stream.from.UnmarshalXMLAttr(attr); err != nil {
				return stream, streamerror.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			stream.id = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			(&stream.version).UnmarshalXMLAttr(attr)
		case xml.Name{Space: "", Local: "xmlns"}:
			if attr.Value != "jabber:client" && attr.Value != "jabber:server" {
				return stream, streamerror.InvalidNamespace
			}
			stream.xmlns = attr.Value
		case xml.Name{Space: "xmlns", Local: "stream"}:
			if attr.Value != ns.Stream {
				return stream, streamerror.InvalidNamespace
			}
		case xml.Name{Space: "xml", Local: "lang"}:
			stream.lang = language.Make(attr.Value)
		}
	}
	return stream, nil
}

// Sends a new XML header followed by a stream start element on the given
// io.Writer. We don't use an xml.Encoder both because Go's standard library xml
// package really doesn't like the namespaced stream:stream attribute and
// because we can guarantee well-formedness of the XML with a print in this case
// and printing is much faster than encoding. Afterwards, clear the
// StreamRestartRequired bit and set the output stream information.
func sendNewStream(w io.Writer, cfg *Config, id string) error {
	stream := stream{
		to:      cfg.Location,
		from:    cfg.Origin,
		lang:    cfg.Lang,
		version: cfg.Version,
	}
	switch cfg.S2S {
	case true:
		stream.xmlns = ns.Server
	case false:
		stream.xmlns = ns.Client
	}

	stream.id = id
	if id == "" {
		id = " "
	} else {
		id = ` id='` + id + `' `
	}

	_, err := fmt.Fprintf(w,
		xmlHeader+`<stream:stream%sto='%s' from='%s' version='%s' xml:lang='%s' xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
		id,
		cfg.Location.String(),
		cfg.Origin.String(),
		cfg.Version,
		cfg.Lang,
		stream.xmlns,
	)
	if err != nil {
		return err
	}

	if conn, ok := w.(*Conn); ok {
		conn.out.stream = stream
	}
	return nil
}

func expectNewStream(ctx context.Context, r io.Reader) error {
	var foundHeader bool

	// If the reader is a Conn, use its decoder, otherwise make a new one.
	var d *xml.Decoder
	if conn, ok := r.(*Conn); ok {
		d = conn.in.d
	} else {
		d = xml.NewDecoder(r)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			switch {
			case tok.Name.Local == "error" && tok.Name.Space == ns.Stream:
				se := streamerror.StreamError{}
				if err := d.DecodeElement(&se, &tok); err != nil {
					return err
				}
				return se
			case tok.Name.Local != "stream":
				return streamerror.BadFormat
			case tok.Name.Space != ns.Stream:
				return streamerror.InvalidNamespace
			}

			stream, err := streamFromStartElement(tok)
			switch {
			case err != nil:
				return err
			case stream.version != internal.DefaultVersion:
				return streamerror.UnsupportedVersion
			}

			if conn, ok := r.(*Conn); ok {
				if (conn.state&Received) != Received && stream.id == "" {
					// if we are the initiating entity and there is no stream ID…
					return streamerror.BadFormat
				}
				conn.in.stream = stream
			}
			return nil
		case xml.ProcInst:
			// TODO: If version or encoding are declared, validate XML 1.0 and UTF-8
			if !foundHeader && tok.Target == "xml" {
				foundHeader = true
				continue
			}
			return streamerror.RestrictedXML
		case xml.EndElement:
			return streamerror.NotWellFormed
		default:
			return streamerror.RestrictedXML
		}
	}
}

func (c *Conn) negotiateStreams(ctx context.Context, rwc io.ReadWriteCloser) (err error) {
	// Loop for as long as we're not done negotiating features or a stream restart
	// is still required.
	for done := false; !done || rwc != nil; {
		if rwc != nil {
			c.features = make(map[xml.Name]struct{})
			c.negotiated = make(map[xml.Name]struct{})
			c.rwc = rwc
			c.in.d = xml.NewDecoder(c.rwc)
			c.out.e = xml.NewEncoder(c.rwc)
			rwc = nil

			if (c.state & Received) == Received {
				// If we're the receiving entity wait for a new stream, then send one in
				// response.
				if err = expectNewStream(ctx, c); err != nil {
					return err
				}
				if err = sendNewStream(c, c.config, internal.RandomID(streamIDLength)); err != nil {
					return err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for one
				// in response.
				if err := sendNewStream(c, c.config, ""); err != nil {
					return err
				}
				if err := expectNewStream(ctx, c); err != nil {
					return err
				}
			}
		}

		if done, rwc, err = c.negotiateFeatures(ctx); err != nil {
			return err
		}
	}
	return nil
}
