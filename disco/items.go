// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// ItemsQuery is the payload of a query for a node's items.
type ItemsQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	Node    string   `xml:"node,attr"`
}

// TokenReader implements xmlstream.Marshaler.
func (q ItemsQuery) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Space: NSItems, Local: "query"}}
	if q.Node != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "node"}, Value: q.Node})
	}
	return xmlstream.Wrap(nil, start)
}

// WriteXML implements xmlstream.WriterTo.
func (q ItemsQuery) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}

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
		Name: xml.Name{Space: NSItems, Local: "item"},
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

// ItemIter is an iterator over discovered items.
type ItemIter struct {
	iter    *xmlstream.Iter
	current Item
	err     error
}

// Next returns true if there are more items to decode.
func (i *ItemIter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	start, r := i.iter.Current()
	d := xml.NewTokenDecoder(r)
	item := Item{}
	i.err = d.DecodeElement(&item, start)
	if i.err != nil {
		return false
	}
	i.current = item
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *ItemIter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Item returns the last roster item parsed by the iterator.
func (i *ItemIter) Item() Item {
	return i.current
}

// Close indicates that we are finished with the given iterator and processing
// the stream may continue.
// Calling it multiple times has no effect.
func (i *ItemIter) Close() error {
	if i.iter == nil {
		return nil
	}
	return i.iter.Close()
}

// GetItemsIQ is like GetItems but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
//
// The iterator must be closed before anything else is done on the session.
// Any errors encountered while creating the iter are deferred until the iter is
// used.
func GetItemsIQ(ctx context.Context, node string, iq stanza.IQ, s *xmpp.Session) *ItemIter {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	query := ItemsQuery{
		Node: node,
	}
	iter, err := s.IterIQ(ctx, iq.Wrap(query.TokenReader()))
	if err != nil {
		return &ItemIter{err: err}
	}
	return &ItemIter{iter: iter}
}
