// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"context"
	"crypto"
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// HandleCaps calls f for each incoming presence containing entity capabilities
// information.
func HandleCaps(f func(stanza.Presence, Caps)) mux.Option {
	return mux.PresenceFunc("", xml.Name{Space: NSCaps, Local: "c"}, func(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
		s := struct {
			stanza.Presence
			Caps Caps
		}{}
		err := xml.NewTokenDecoder(r).Decode(&s)
		if err != nil {
			return err
		}
		f(p, s.Caps)
		return nil
	})
}

// StreamFeature is an informational stream feature that saves any entity caps
// information that was published by the server during session negotiation.
// StreamFeature should not be used on the server side.
func StreamFeature() xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name: xml.Name{Space: NSCaps, Local: "c"},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			c := Caps{}
			err := d.DecodeElement(&c, start)
			return false, c, err
		},
	}
}

// ServerCaps returns any entity caps information advertised by the server when
// we first connected.
// If the ServerCaps feature was not used during session negotiation or no
// entity caps was advertised when connecting, ok will be false.
func ServerCaps(s *xmpp.Session) (c Caps, ok bool) {
	data, advertised := s.Feature(NSCaps)
	c, ok = data.(Caps)
	return c, ok && advertised
}

// Caps can be included in a presence stanza or in stream features to advertise
// entity capabilities.
// Node is a string that uniquely identifies your client (eg.
// https://example.com/myclient) and ver is the hash of an Info value.
type Caps struct {
	XMLName xml.Name    `xml:"http://jabber.org/protocol/caps c"`
	Hash    crypto.Hash `xml:"hash,attr"`
	Node    string      `xml:"node,attr"`
	Ver     string      `xml:"ver,attr"`
}

// TokenReader implements xmlstream.Marshaler.
func (c Caps) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(nil, xml.StartElement{
		Name: xml.Name{Space: NSCaps, Local: "c"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "hash"}, Value: strings.ToLower(c.Hash.String())},
			{Name: xml.Name{Local: "node"}, Value: c.Node},
			{Name: xml.Name{Local: "ver"}, Value: c.Ver},
		},
	})
}

// WriteXML implements xmlstream.WriterTo.
func (c Caps) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, c.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (c Caps) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := c.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler
func (c *Caps) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "hash":
			switch attr.Value {
			case "sha-1":
				c.Hash = crypto.SHA1
			case "sha-224":
				c.Hash = crypto.SHA224
			case "sha-256":
				c.Hash = crypto.SHA256
			case "sha-384":
				c.Hash = crypto.SHA384
			case "sha-512":
				c.Hash = crypto.SHA512
			}
		case "node":
			c.Node = attr.Value
		case "ver":
			c.Ver = attr.Value
		}
	}
	return xmlstream.Skip(d)
}
