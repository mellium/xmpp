// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster implements contact list functionality.
package roster // import "mellium.im/xmpp/roster"

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/iter"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package provided as a convenience.
const (
	NS = "jabber:iq:roster"
)

// Iter is an iterator over roster items.
type Iter struct {
	iter    *iter.Iter
	current Item
	err     error
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil {
		return false
	}
	n := i.iter.Next()
	if !n {
		return false
	}
	start, r := i.iter.Current()
	d := xml.NewTokenDecoder(xmlstream.Wrap(r, start.Copy()))
	item := Item{}
	i.err = d.Decode(&item)
	if i.err != nil {
		return false
	}
	i.current = item
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Item returns the last roster item parsed by the iterator.
func (i *Iter) Item() Item {
	return i.current
}

// Close indicates that we are finished with the given iterator and processing
// the stream may continue.
// Calling it multiple times has no effect.
func (i *Iter) Close() error {
	return i.iter.Close()
}

// Fetch requests the roster and returns an iterator over all roster items
// (blocking until a response is received).
//
// The iterator must be closed before anything else is done on the session or it
// will become invalid.
// Any errors encountered while creating the iter are deferred until the iter is
// used.
func Fetch(ctx context.Context, s *xmpp.Session) *Iter {
	return FetchIQ(ctx, stanza.IQ{}, s)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) *Iter {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	rosterIQ := IQ{IQ: iq}
	payload := rosterIQ.payload()
	r, err := s.SendIQElement(ctx, payload, iq)
	if err != nil {
		return &Iter{err: err}
	}

	// Pop the start IQ token.
	_, err = r.Token()
	if err != nil {
		return &Iter{err: err}
	}

	// Pop the roster wrapper token.
	_, err = r.Token()
	if err != nil {
		return &Iter{err: err}
	}

	// Return the iterator which will parse the rest of the payload incrementally.
	return &Iter{
		iter: iter.New(r),
	}
}

// IQ represents a user roster request or response.
// The zero value is a valid query for the roster.
type IQ struct {
	stanza.IQ

	Query struct {
		Ver  string `xml:"version,attr,omitempty"`
		Item []Item `xml:"item"`
	} `xml:"jabber:iq:roster query"`
}

type itemMarshaler struct {
	items []Item
	cur   xml.TokenReader
}

func (m itemMarshaler) Token() (xml.Token, error) {
	if len(m.items) == 0 {
		return nil, io.EOF
	}

	if m.cur == nil {
		var item Item
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

// TokenReader returns a stream of XML tokens that match the IQ.
func (iq IQ) TokenReader() xml.TokenReader {
	if iq.IQ.Type != stanza.GetIQ {
		iq.IQ.Type = stanza.GetIQ
	}

	return stanza.WrapIQ(iq.IQ, iq.payload())
}

// Payload returns a stream of XML tokekns that match the roster query payload
// without the IQ wrapper.
func (iq IQ) payload() xml.TokenReader {
	attrs := []xml.Attr{}
	if iq.Query.Ver != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: iq.Query.Ver})
	}

	return xmlstream.Wrap(
		itemMarshaler{items: iq.Query.Item},
		xml.StartElement{Name: xml.Name{Local: "query", Space: NS}, Attr: attrs},
	)
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

// Item represents a contact in the roster.
type Item struct {
	JID          jid.JID `xml:"jid,attr,omitempty"`
	Name         string  `xml:"name,attr,omitempty"`
	Subscription string  `xml:"subscription,attr,omitempty"`
	Group        string  `xml:"group,omitempty"`
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (item Item) TokenReader() xml.TokenReader {
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
func (item Item) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, item.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (item Item) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := item.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}
