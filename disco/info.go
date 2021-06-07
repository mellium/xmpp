// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// InfoQuery is the payload of a query for a node's identities and features.
type InfoQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	Node    string   `xml:"node,attr,omitempty"`
}

func (q InfoQuery) wrap(r xml.TokenReader) xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Space: NSInfo, Local: "query"}}
	if q.Node != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "node"}, Value: q.Node})
	}
	return xmlstream.Wrap(r, start)
}

// TokenReader implements xmlstream.Marshaler.
func (q InfoQuery) TokenReader() xml.TokenReader {
	return q.wrap(nil)
}

// WriteXML implements xmlstream.WriterTo.
func (q InfoQuery) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}

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
		Name: xml.Name{Space: NSInfo, Local: "query"},
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

// Info is a response to a disco info query.
type Info struct {
	InfoQuery
	Identity []Identity     `xml:"identity"`
	Features []info.Feature `xml:"feature"`
	Form     *form.Data     `xml:"jabber:x:data x,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (i Info) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	for _, f := range i.Features {
		payloads = append(payloads, f.TokenReader())
	}
	for _, ident := range i.Identity {
		payloads = append(payloads, ident.TokenReader())
	}
	return i.InfoQuery.wrap(xmlstream.MultiReader(payloads...))
}

// WriteXML implements xmlstream.WriterTo.
func (i Info) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// GetInfo discovers a set of features and identities associated with a JID and
// optional node.
// An empty Node means to query the root items for the JID.
// It blocks until a response is received.
func GetInfo(ctx context.Context, node string, to jid.JID, s *xmpp.Session) (Info, error) {
	return GetInfoIQ(ctx, node, stanza.IQ{To: to}, s)
}

// GetInfoIQ is like GetInfo but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func GetInfoIQ(ctx context.Context, node string, iq stanza.IQ, s *xmpp.Session) (Info, error) {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	query := InfoQuery{
		Node: node,
	}
	var info Info
	err := s.UnmarshalIQElement(ctx, query.TokenReader(), iq, &info)
	return info, err
}
