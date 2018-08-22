// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster implements contact list functionality.
package roster

import (
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package provided as a convenience.
const (
	NS = "jabber:iq:roster"
)

// IQ represents a user roster request or response.
// The zero value is a valid query for the roster.
type IQ struct {
	stanza.IQ

	Query struct {
		Ver  string       `xml:"version,attr,omitempty"`
		Item []RosterItem `xml:"item"`
	} `xml:"jabber:iq:roster query"`
}

type itemMarshaler struct {
	items []RosterItem
	cur   xml.TokenReader
}

func (m itemMarshaler) Token() (xml.Token, error) {
	if len(m.items) == 0 {
		return nil, io.EOF
	}

	if m.cur == nil {
		var item RosterItem
		item, m.items = m.items[0], m.items[1:]
		m.cur = item.TokenReader()
	}

	tok, err := m.cur.Token()
	if err != nil && err != io.EOF {
		return tok, err
	}

	if tok == nil {
		m.cur = nil
		return m.Token()
	}

	return tok, nil
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (iq IQ) TokenReader() xml.TokenReader {
	attrs := []xml.Attr{}
	if iq.Query.Ver != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: iq.Query.Ver})
	}
	if iq.IQ.Type != stanza.GetIQ {
		iq.IQ.Type = stanza.GetIQ
	}

	return stanza.WrapIQ(&iq.IQ, xmlstream.Wrap(
		itemMarshaler{items: iq.Query.Item},
		xml.StartElement{Name: xml.Name{Local: "query", Space: NS}, Attr: attrs},
	))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (iq IQ) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (iq IQ) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := iq.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// RosterItem represents a contact in the roster.
type RosterItem struct {
	JID          jid.JID `xml:"jid,attr,omitempty"`
	Name         string  `xml:"name,attr,omitempty"`
	Subscription string  `xml:"subscription,attr,omitempty"`
	Group        string  `xml:"group,omitempty"`
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (item RosterItem) TokenReader() xml.TokenReader {
	var group xml.TokenReader
	if item.Group != "" {
		group = xmlstream.Wrap(
			xmlstream.Token(xml.CharData(item.Group)),
			xml.StartElement{
				Name: xml.Name{Local: "group"},
			},
		)
	}

	attrs := []xml.Attr{}
	if j := item.JID.String(); j != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "jid"}, Value: j})
	}
	if item.Name != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "name"}, Value: item.Name})
	}
	if item.Subscription != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "subscription"}, Value: item.Subscription})
	}

	return xmlstream.Wrap(
		group,
		xml.StartElement{
			Name: xml.Name{Local: "item"},
			Attr: attrs,
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (item RosterItem) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, item.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (item RosterItem) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := item.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}
