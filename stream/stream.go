// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream

import (
	"encoding/xml"

	"mellium.im/xmpp/jid"
)

// Info contains metadata extracted from a stream start token.
type Info struct {
	Name    xml.Name
	XMLNS   string
	To      jid.JID
	From    jid.JID
	ID      string
	Version Version
	Lang    string
}

// FromStartElement sets the data in Info from the provided StartElement.
func (i *Info) FromStartElement(s xml.StartElement) error {
	i.Name = s.Name
	for _, attr := range s.Attr {
		switch attr.Name {
		case xml.Name{Space: "", Local: "xmlns"}:
			i.XMLNS = attr.Value
		case xml.Name{Space: "", Local: "to"}:
			if err := (&i.To).UnmarshalXMLAttr(attr); err != nil {
				return ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			if err := (&i.From).UnmarshalXMLAttr(attr); err != nil {
				return ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			i.ID = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			err := (&i.Version).UnmarshalXMLAttr(attr)
			if err != nil {
				return BadFormat
			}
		case xml.Name{Space: "xml", Local: "lang"}:
			i.Lang = attr.Value
		}
	}
	return nil
}
