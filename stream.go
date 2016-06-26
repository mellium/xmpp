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
)

type stream struct {
	to      *jid.JID         `xml:"to,attr"`
	from    *jid.JID         `xml:"from,attr"`
	id      string           `xml:"id,attr,ommitempty"`
	version internal.Version `xml:"version,attr,ommitempty"`
	xmlns   string           `xml:"xmlns,attr"`
	lang    language.Tag     `xml:"http://www.w3.org/XML/1998/namespace lang"`
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
func sendNewStream(w io.Writer, c *Config, id string) error {
	stream := stream{
		to:      c.Location,
		from:    c.Origin,
		lang:    c.Lang,
		version: c.Version,
	}
	switch c.S2S {
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
		c.Location.String(),
		c.Origin.String(),
		c.Version,
		c.Lang,
		stream.xmlns,
	)
	if err != nil {
		return err
	}

	// Clear the StreamRestartRequired bit
	if c, ok := w.(*Conn); ok {
		c.state &= ^StreamRestartRequired
		c.out = stream
	}
	return err
}

// Fetch a token from the given decoder. If it is not a new stream start element
// (or an XML header followed by a stream), error. Clear the
// StreamRestartRequired bit afterwards.
func expectNewStream(ctx context.Context, d *xml.Decoder, c *Conn) error {
	var foundHeader bool
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
				return BadNamespacePrefix
			}

			stream, err := streamFromStartElement(tok)
			if err != nil {
				return err
			}
			if stream.version != internal.DefaultVersion {
				return UnsupportedVersion
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

func (c *Conn) negotiateStreams(ctx context.Context) error {
	panic("xmpp: Not yet implemented.")
}
