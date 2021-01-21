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
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// Info contains metadata extracted from a stream start token.
type Info struct {
	Name    xml.Name
	to      *jid.JID
	from    *jid.JID
	ID      string
	version Version
	xmlns   string
	lang    string
}

// This MUST only return stream errors.
// TODO: Is the above true? Just make it return a StreamError?
func streamFromStartElement(s xml.StartElement, ws bool) (Info, error) {
	streamData := Info{
		Name: s.Name,
	}
	for _, attr := range s.Attr {
		switch attr.Name {
		case xml.Name{Space: "", Local: "to"}:
			streamData.to = &jid.JID{}
			if err := streamData.to.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			streamData.from = &jid.JID{}
			if err := streamData.from.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			streamData.ID = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			err := (&streamData.version).UnmarshalXMLAttr(attr)
			if err != nil {
				return streamData, stream.BadFormat
			}
		case xml.Name{Space: "", Local: "xmlns"}:
			// TODO:  why did we do this in the case block? Shouldn't we check s.Name?
			if (ws && attr.Value != ns.WS) || (!ws && attr.Value != "jabber:client" && attr.Value != "jabber:server") {
				return streamData, fmt.Errorf("xmpp: invalid xmlns attribute: %s", attr.Value)
			}
			streamData.xmlns = attr.Value
		case xml.Name{Space: "xmlns", Local: "stream"}:
			// If we're using the websocket subprotocol this will never show up (but
			// if it does, we don't care at all, it's just extra stuff that we won't
			// end up using).
			if !ws && attr.Value != stream.NS {
				return streamData, stream.InvalidNamespace
			}
		case xml.Name{Space: "xml", Local: "lang"}:
			streamData.lang = attr.Value
		}
	}
	return streamData, nil
}

// Send sends a new XML header followed by a stream start element on the given
// io.Writer.
// We don't use an xml.Encoder both because Go's standard library xml package
// really doesn't like the namespaced stream:stream attribute and because we can
// guarantee well-formedness of the XML with a print in this case and printing
// is much faster than encoding.
// Afterwards, clear the StreamRestartRequired bit and set the output stream
// information.
func Send(rw io.ReadWriter, s2s, ws bool, version Version, lang string, location, origin, id string) (Info, error) {
	streamData := Info{}
	switch s2s {
	case true:
		streamData.xmlns = ns.Server
	case false:
		streamData.xmlns = ns.Client
	}

	streamData.ID = id
	if id != "" {
		id = `id='` + id + `' `
	}

	b := bufio.NewWriter(rw)
	var err error
	if ws {
		_, err = fmt.Fprintf(b,
			`<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" %sto='%s' from='%s' version='%s'`,
			id,
			location,
			origin,
			version,
		)
	} else {
		_, err = fmt.Fprintf(b,
			decl.XMLHeader+`<stream:stream %sto='%s' from='%s' version='%s'`,
			id,
			location,
			origin,
			version,
		)
	}
	if err != nil {
		return streamData, err
	}

	if len(lang) > 0 {
		_, err = b.Write([]byte(" xml:lang='"))
		if err != nil {
			return streamData, err
		}
		err = xml.EscapeText(b, []byte(lang))
		if err != nil {
			return streamData, err
		}
		_, err = b.Write([]byte("'"))
		if err != nil {
			return streamData, err
		}
	}

	if ws {
		_, err = fmt.Fprint(b, `/>`)
	} else {
		_, err = fmt.Fprintf(b, ` xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
			streamData.xmlns,
		)
	}
	if err != nil {
		return streamData, err
	}

	return streamData, b.Flush()
}

// Expect reads a token from d and expects that it will be a new stream start
// token.
// If not, an error is returned. It then handles feature negotiation for the new
// stream.
// If an XML header is discovered instead, it is skipped.
func Expect(ctx context.Context, d xml.TokenReader, recv, ws bool) (streamData Info, err error) {
	// Skip the XML declaration (if any).
	d = decl.Skip(d)

	for {
		select {
		case <-ctx.Done():
			return streamData, ctx.Err()
		default:
		}
		t, err := d.Token()
		if err != nil {
			return streamData, err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			switch {
			case tok.Name.Local == "error" && tok.Name.Space == stream.NS:
				se := stream.Error{}
				if err := xml.NewTokenDecoder(d).DecodeElement(&se, &tok); err != nil {
					return streamData, err
				}
				return streamData, se
			case !ws && tok.Name.Local != "stream":
				// TODO: return sane error.
				return streamData, stream.BadFormat
			case ws && tok.Name.Local != "open":
				// TODO: return sane error.
				return streamData, stream.BadFormat
			case !ws && tok.Name.Space != stream.NS:
				// TODO: send invalid namespace, return sane error.
				return streamData, fmt.Errorf("xmpp: invalid stream namespace: %s", tok.Name.Space)
			case ws && tok.Name.Space != ns.WS:
				// TODO: send invalid namespace, return sane error.
				return streamData, fmt.Errorf("xmpp: invalid WebSocket stream namespace: %s", tok.Name.Space)
			case ws && tok.Name.Local == "open" && tok.Name.Space == ns.WS:
				// Websocket payloads are always full XML documents, so the "open"
				// element is closed as well.
				err = xmlstream.Skip(d)
				if err != nil {
					return streamData, err
				}
			}

			streamData, err = streamFromStartElement(tok, ws)
			switch {
			case err != nil:
				return streamData, err
			case streamData.version != DefaultVersion:
				return streamData, stream.UnsupportedVersion
			}

			if !recv && streamData.ID == "" {
				// if we are the initiating entity and there is no stream ID…
				return streamData, stream.BadFormat
			}
			return streamData, nil
		case xml.ProcInst:
			return streamData, stream.RestrictedXML
		case xml.EndElement:
			return streamData, stream.NotWellFormed
		default:
			return streamData, stream.RestrictedXML
		}
	}
}
