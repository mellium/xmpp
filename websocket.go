// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/jid"
	"golang.org/x/net/websocket"
	"golang.org/x/text/language"
)

// Open is a struct that can be marshaled and unmarshaled into a valid
// <open/> element to start a websockets session.
type Open struct {
	XMLName xml.Name     `xml:"urn:ietf:params:xml:ns:xmpp-framing open"`
	To      jid.JID      `xml:"to,attr"`
	From    jid.JID      `xml:"from,attr,omitempty"`
	Version string       `xml:"version,attr,omitempty"`
	Lang    language.Tag `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Id      string       `xml:"id,attr,omitempty"`
}

func (o *Open) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return nil
}

func (o *Open) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.Encode(struct {
		Open

		Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	}{
		Open: *o,
		Lang: o.Lang.String(),
	})
}

// Close is a struct that can be marshaled and unmarshaled into a valid
// <close/> element to end a websockets session.
type Close struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-framing close"`
}

// Codec returns a websocket codec for framing an XMPP session that is RFC 7395
// compliant.
func Codec() websocket.Codec {
	return websocket.Codec{
		Marshal: func(v interface{}) (data []byte, payloadType byte, err error) {
			return []byte{}, 0, nil
		},
		Unmarshal: func(data []byte, payloadType byte, v interface{}) (err error) {
			return nil
		},
	}
}
