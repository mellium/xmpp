// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

// InfoQuery is the payload of a query for a node's identities and features.
type InfoQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	Node    string   `xml:"node,attr,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (q InfoQuery) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Space: NSInfo, Local: "query"}}
	if q.Node != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "node"}, Value: q.Node})
	}
	return xmlstream.Wrap(nil, start)
}

// WriteXML implements xmlstream.WriterTo.
func (q InfoQuery) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}
