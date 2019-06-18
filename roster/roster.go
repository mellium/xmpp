// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster implements contact list functionality.
package roster

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package provided as a convenience.
const (
	NS = "jabber:iq:roster"
)

// Iter is an iterator over roster items.
type Iter struct {
	r    xmlstream.TokenReadCloser
	d    *xml.Decoder
	err  error
	item Item
	next *xml.StartElement
}

func (i *Iter) setNext() {
	i.next = nil
	t, err := i.d.Token()
	if err != nil {
		i.err = err
		return
	}
	start, ok := t.(xml.StartElement)
	// If we're done with the items and get the roster payload end element (or
	// anything else), call it done.
	if !ok {
		return
	}

	// TODO: check the name of the payload to make sure the server is behaving
	// correctly.

	i.next = &start
}

// Returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || i.next == nil {
		return false
	}

	// If this is a start element, decode the item.
	item := Item{}
	err := i.d.DecodeElement(&item, i.next)
	if err != nil {
		i.err = err
		return false
	}
	i.item = item
	ret := i.err == nil && i.next != nil
	i.setNext()
	return ret
}

// Returns the current roster item.
func (i *Iter) Item() Item {
	return i.item
}

// Returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	return i.err
}

// Close indicates that we are finished with the given roster.
// Calling it multiple times has no effect.
func (i *Iter) Close() error {
	i.next = nil
	return i.r.Close()
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
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) *Iter {
	rosterIQ := IQ{IQ: iq}
	r, err := s.Send(ctx, rosterIQ.TokenReader())
	if err != nil {
		return &Iter{err: err}
	}

	d := xml.NewTokenDecoder(r)

	// Pop the start IQ token.
	_, err = d.Token()
	if err != nil {
		return &Iter{err: err}
	}

	// Pop the roster wrapper token.
	_, err = d.Token()
	if err != nil {
		return &Iter{err: err}
	}

	// Return the iterator which will parse the rest of the payload incrementally.
	iter := &Iter{
		r: r,
		d: d,
	}
	iter.setNext()
	return iter
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
