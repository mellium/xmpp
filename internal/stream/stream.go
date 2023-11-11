// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package stream contains internal stream parsing and handling behavior.
package stream // import "mellium.im/xmpp/internal/stream"

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/decl"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

const wsNamespace = "urn:ietf:params:xml:ns:xmpp-framing"

// Send sends a new XML header followed by a stream start element on the given
// io.Writer.
// We don't use an xml.Encoder both because Go's standard library xml package
// really doesn't like the namespaced stream:stream attribute and because we can
// guarantee well-formedness of the XML with a print in this case and printing
// is much faster than encoding.
// Afterwards, clear the StreamRestartRequired bit and set the output stream
// information.
func Send(rw io.ReadWriter, streamData *stream.Info, ws bool, version stream.Version, lang, to, from, id string) error {
	streamData.ID = id
	b := bufio.NewWriter(rw)
	var err error
	if ws {
		_, err = fmt.Fprintf(b,
			`<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" version='%s'`,
			version,
		)
	} else {
		_, err = fmt.Fprintf(b,
			decl.XMLHeader+`<stream:stream xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams' version='%s'`,
			streamData.XMLNS,
			version,
		)
	}
	if err != nil {
		return err
	}

	if id != "" {
		_, err = fmt.Fprintf(b, " id='%s'", id)
		if err != nil {
			return err
		}
	}
	if to != "" {
		_, err = fmt.Fprintf(b, " to='%s'", to)
		if err != nil {
			return err
		}
	}
	if from != "" {
		_, err = fmt.Fprintf(b, " from='%s'", from)
		if err != nil {
			return err
		}
	}

	if len(lang) > 0 {
		_, err = b.Write([]byte(" xml:lang='"))
		if err != nil {
			return err
		}
		err = xml.EscapeText(b, []byte(lang))
		if err != nil {
			return err
		}
		_, err = b.Write([]byte("'"))
		if err != nil {
			return err
		}
	}

	if ws {
		_, err = fmt.Fprint(b, `/>`)
	} else {
		_, err = fmt.Fprint(b, `>`)
	}
	if err != nil {
		return err
	}

	return b.Flush()
}

// Expect reads a token from d and expects that it will be a new stream start
// token.
// If not, an error is returned. It then handles feature negotiation for the new
// stream.
// If an XML header is discovered instead, it is skipped.
func Expect(ctx context.Context, in *stream.Info, d xml.TokenReader, recv, ws bool) error {
	// Skip the XML declaration (if any).
	d = negotiateReader(decl.Skip(d), ws)

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
		case xml.CharData:
			// If we get whitespace (the only valid chardata let through by the
			// negotiateReader call above), skip it.
			continue
		case xml.StartElement:
			switch {
			case tok.Name.Local == "error" && tok.Name.Space == stream.NS:
				se := stream.Error{}
				if err := xml.NewTokenDecoder(d).DecodeElement(&se, &tok); err != nil {
					return err
				}
				return se
			case !ws && (tok.Name.Local != "stream" || tok.Name.Space != stream.NS):
				return fmt.Errorf("expected stream open element %v, got %v: %w", xml.Name{Space: stream.NS, Local: "stream"}, tok.Name, stream.InvalidNamespace)
			case ws && (tok.Name.Local != "open" || tok.Name.Space != wsNamespace):
				return fmt.Errorf("expected WebSocket stream open element %v, got %v: %w", xml.Name{Space: wsNamespace, Local: "open"}, tok.Name, stream.InvalidNamespace)
			case ws && tok.Name.Local == "open" && tok.Name.Space == wsNamespace:
				// Websocket payloads are always full XML documents, so the "open"
				// element is closed as well.
				err = xmlstream.Skip(d)
				if err != nil {
					return err
				}
			}

			err = in.FromStartElement(tok)
			switch {
			case err != nil:
				return err
			case in.Version != stream.DefaultVersion:
				return stream.UnsupportedVersion
			}

			if !ws && in.XMLNS != stanza.NSClient && in.XMLNS != stanza.NSServer {
				return fmt.Errorf("expected jabber:client or jabber:server for default namespace, got %q: %w", in.XMLNS, stream.InvalidNamespace)
			}

			if !recv && in.ID == "" {
				// if we are the initiating entity and there is no stream ID…
				return fmt.Errorf("initiating entity must set stream ID: %w", stream.BadFormat)
			}
			return nil
		}
	}
}

const (
	closeStreamTag   = `</stream:stream>`
	closeStreamWSTag = `<close xmlns="urn:ietf:params:xml:ns:xmpp-framing"/>`
)

// Close sends a stream end token.
func Close(w io.Writer, streamData *stream.Info) error {
	var err error
	switch xmlns := streamData.Name.Space; xmlns {
	case wsNamespace:
		_, err = w.Write([]byte(closeStreamWSTag))
	default:
		// case stream.NS:
		_, err = w.Write([]byte(closeStreamTag))
	}
	return err
}
