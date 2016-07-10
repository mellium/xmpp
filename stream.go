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
)

const streamIDLength = 16

// SessionState represents the current state of an XMPP session. For a
// description of each bit, see the various SessionState typed constants.
type SessionState int8

const (
	// Indicates that the underlying connection has been secured. For instance,
	// after STARTTLS has been performed or if a pre-secured connection is being
	// used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Indicates that the session has been authenticated (probably with SASL).
	Authn

	// Indicates that an XMPP resource has been bound.
	Bind

	// Indicates that the session is fully negotiated and that XMPP stanzas may be
	// sent and received.
	Ready

	// Indicates that the session's streams must be restarted. This bit will
	// trigger an automatic restart and will be flipped back to off as soon as the
	// stream is restarted.
	StreamRestartRequired

	// Indicates that the session was initiated by a foreign entity.
	Received

	// Indicates that the stream should be (or has been) terminated. After being
	// flipped, this bit is left off unless the stream is restarted. This does not
	// provide any information about the underlying TLS connection.
	EndStream
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
				return stream, ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			stream.from = &jid.JID{}
			if err := stream.from.UnmarshalXMLAttr(attr); err != nil {
				return stream, ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			stream.id = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			(&stream.version).UnmarshalXMLAttr(attr)
		case xml.Name{Space: "", Local: "xmlns"}:
			if attr.Value != "jabber:client" && attr.Value != "jabber:server" {
				return stream, InvalidNamespace
			}
			stream.xmlns = attr.Value
		case xml.Name{Space: "xmlns", Local: "stream"}:
			if attr.Value != NSStream {
				return stream, InvalidNamespace
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
		stream.xmlns = NSServer
	case false:
		stream.xmlns = NSClient
	}

	stream.id = id
	if id == "" {
		id = " "
	} else {
		id = ` id='` + id + `' `
	}

	_, err := fmt.Fprint(w, xml.Header)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w,
		`<stream:stream%sto='%s' from='%s' version='%s' xml:lang='%s' xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
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
		conn.state &= ^StreamRestartRequired
		conn.out.stream = stream
		conn.out.e = xml.NewEncoder(w)
	}
	return nil
}

func expectNewStream(ctx context.Context, r io.Reader) error {
	var foundHeader bool
	var d *xml.Decoder
	if conn, ok := r.(*Conn); ok {
		if conn.in.d == nil {
			conn.in.d = xml.NewDecoder(r)
		}
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
		t, err := d.RawToken()
		if err != nil {
			return err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			switch {
			case tok.Name.Local != "stream":
				return BadFormat
			case tok.Name.Space != "stream":
				return InvalidNamespace
			}

			stream, err := streamFromStartElement(tok)
			switch {
			case err != nil:
				return err
			case stream.version != internal.DefaultVersion:
				return UnsupportedVersion
			}

			if conn, ok := r.(*Conn); ok {
				if (conn.state&Received) != Received && stream.id == "" {
					// if we are the initiating entity and there is no stream ID…
					return BadFormat
				}
				conn.state &= ^StreamRestartRequired
				conn.in.stream = stream
				conn.in.d = xml.NewDecoder(r)
			}
			return nil
		case xml.ProcInst:
			// TODO: If version or encoding are declared, validate XML 1.0 and UTF-8
			if !foundHeader && tok.Target == "xml" {
				foundHeader = true
				continue
			}
			return RestrictedXML
		case xml.EndElement:
			return NotWellFormed
		default:
			return RestrictedXML
		}
	}
}

func (c *Conn) negotiateStreams(ctx context.Context) (err error) {
	if (c.state & Received) == Received {
		if err = expectNewStream(ctx, c); err != nil {
			return err
		}
		if err = sendNewStream(c, c.config, internal.RandomID(streamIDLength)); err != nil {
			return err
		}
	} else {
		if err := sendNewStream(c, c.config, ""); err != nil {
			return err
		}
		if err := expectNewStream(ctx, c); err != nil {
			return err
		}
	}

	for done := false; !done; done, err = c.negotiateFeatures(ctx) {
		switch {
		case err != nil:
			return err
		case c.state&StreamRestartRequired == StreamRestartRequired:
			// If we require a stream restart, do so…

			// BUG(ssw): Negotiating streams can lead to a stack overflow when
			//           connecting to a malicious endpoint.
			return c.negotiateStreams(ctx)
		}
	}
	panic("xmpp: Not yet implemented.")
}
