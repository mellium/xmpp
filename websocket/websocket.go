// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package websocket provides a codec that implements the websocket subprotocol
// for XMPP defined in RFC 7395.
package websocket // import "mellium.im/xmpp/websocket"

import (
	"encoding/xml"

	"mellium.im/xmpp/codec"
	"mellium.im/xmpp/ns"
)

// Namespaces used by the websocket subprotocol.
const (
	NS = "urn:ietf:params:xml:ns:xmpp-framing"
)

type websocketTokenReader struct {
	d    *xml.Decoder
	next xml.Token
}

func (wtr *websocketTokenReader) RawToken() (t xml.Token, err error) {
	if wtr.next != nil {
		t = wtr.next
		wtr.next = nil
		return
	}
	t, err = wtr.d.RawToken()
	if err != nil {
		return
	}

	switch tok := t.(type) {
	case xml.StartElement:
		// Turn <stream:stream>'s into <open/>'s
		if tok.Name.Local == "stream" && tok.Name.Space != "" {
			prefix := tok.Name.Space
			tok.Name.Local = "open"
			tok.Name.Space = NS
			newattrs := tok.Attr[:0]
			for _, a := range tok.Attr {
				switch {
				case a.Name.Local == "xmlns" && a.Name.Space == "":
					a.Value = NS
				case a.Name.Space == "xmlns":
					if a.Name.Local == prefix && a.Value != ns.Stream {
						// Oops, this isn't actually a proper <stream:stream/>, it has a
						// different namespace!
						return t, nil
					}
					continue
				}
				newattrs = append(newattrs, a)
			}
			tok.Attr = newattrs
			wtr.next = tok.End()
			return tok, nil
		}
	case xml.EndElement:
		// Turn </stream:stream>'s into <close/>'s
		if tok.Name.Local == "stream" && tok.Name.Space == "stream" {
			start := xml.StartElement{Name: xml.Name{Local: "close", Space: NS}}
			wtr.next = start.End()
			return start, nil
		}
	}
	return
}

func (wtr *websocketTokenReader) Skip() error {
	// Make sure skip works for <open/> and <close/>.
	if wtr.next != nil {
		wtr.next = nil
		return nil
	}

	return wtr.d.Skip()
}

// NewCodec
func NewCodec(d *xml.Decoder, e *xml.Encoder) codec.Codec {
	return struct {
		*xml.Decoder
		*xml.Encoder
	}{
		Decoder: xml.NewTokenDecoder(&websocketTokenReader{
			d: d,
		}),
	}
}
