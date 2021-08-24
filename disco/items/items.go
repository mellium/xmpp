// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package items contains service discovery items.
//
// These were separated out into a separate package to prevent import loops.
package items // import "mellium.im/xmpp/disco/items"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

const (
	ns = `http://jabber.org/protocol/disco#items`
)

// Item represents a discovered item.
type Item struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items item"`
	JID     jid.JID  `xml:"jid,attr"`
	Name    string   `xml:"name,attr,omitempty"`
	Node    string   `xml:"node,attr,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (i Item) TokenReader() xml.TokenReader {
	start := xml.StartElement{
		Name: xml.Name{Space: ns, Local: "item"},
		Attr: []xml.Attr{{
			Name:  xml.Name{Local: "jid"},
			Value: i.JID.String(),
		}},
	}
	if i.Node != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "node"}, Value: i.Node})
	}
	if i.Name != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "name"}, Value: i.Name})
	}
	return xmlstream.Wrap(nil, start)
}

// WriteXML implements xmlstream.WriterTo.
func (i Item) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (i Item) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := i.WriteXML(e)
	return err
}

// Iter is the interface implemented by types that respond to service discovery
// requests for items.
type Iter interface {
	ForItems(node string, f func(Item) error) error
}
