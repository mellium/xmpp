// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package info

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
)

// Identity is the type and category of a node on the network.
// Normally one of the pre-defined Identity types should be used.
type Identity struct {
	XMLName  xml.Name `xml:"http://jabber.org/protocol/disco#info identity"`
	Category string   `xml:"category,attr"`
	Type     string   `xml:"type,attr"`
	Name     string   `xml:"name,attr,omitempty"`
	Lang     string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (i Identity) TokenReader() xml.TokenReader {
	start := xml.StartElement{
		Name: xml.Name{Space: nsInfo, Local: "identity"},
		Attr: []xml.Attr{{
			Name:  xml.Name{Local: "category"},
			Value: i.Category,
		}, {
			Name:  xml.Name{Local: "type"},
			Value: i.Type,
		}},
	}
	if i.Name != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name: xml.Name{Local: "name"}, Value: i.Name,
		})
	}
	if i.Lang != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: i.Lang,
		})
	}
	return xmlstream.Wrap(nil, start)
}

// WriteXML implements xmlstream.WriterTo.
func (i Identity) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (i Identity) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := i.WriteXML(e)
	return err
}

// IdentityIter is the interface implemented by types that implement disco
// identities.
type IdentityIter interface {
	ForIdentities(node string, f func(Identity) error) error
}
