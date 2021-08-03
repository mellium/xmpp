// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster implements contact list functionality.
package roster // import "mellium.im/xmpp/roster"

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package provided as a convenience.
const (
	NS         = "jabber:iq:roster"
	NSFeatures = "urn:xmpp:features:rosterver"
)

// Handle returns an option that registers a Handler for roster pushes.
func Handle(h Handler) mux.Option {
	return mux.IQ(stanza.SetIQ, xml.Name{Local: "query", Space: NS}, h)
}

// Handler responds to roster pushes.
// If Push returns a stanza.Error it is sent as an error response to the IQ
// push, otherwise it is passed through and returned from HandleIQ.
type Handler struct {
	Push func(ver string, item Item) error
}

// HandleIQ responds to roster push IQs.
func (h Handler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	item := Item{}
	err := xml.NewTokenDecoder(t).Decode(&item)
	if err != nil {
		return err
	}
	var ver string
	for _, attr := range start.Attr {
		if attr.Name.Local == "ver" {
			ver = attr.Value
			break
		}
	}
	err = h.Push(ver, item)
	var stanzaErr stanza.Error
	isStanzaErr := errors.As(err, &stanzaErr)
	if isStanzaErr {
		_, err = xmlstream.Copy(t, iq.Error(stanzaErr))
		return err
	}
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(t, iq.Result(nil))
	return err
}

// Iter is an iterator over roster items.
type Iter struct {
	iter    *xmlstream.Iter
	current Item
	err     error
	ver     string
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	start, r := i.iter.Current()
	// If we encounter a lone token that doesn't begin with a start element (eg.
	// a comment) skip it. This should never happen with XMPP, but we don't want
	// to panic in case this somehow happens so just skip it.
	if start == nil {
		return i.Next()
	}
	d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), r))
	item := Item{}
	i.err = d.Decode(&item)
	if i.err != nil {
		return false
	}
	i.current = item
	return true
}

// Version returns the roster version being iterated over or the empty string if
// roster versioning is not enabled.
func (i *Iter) Version() string {
	return i.ver
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
	if i.iter == nil {
		return nil
	}
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
	return FetchIQ(ctx, IQ{}, s)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ or adding items has no effect.
func FetchIQ(ctx context.Context, iq IQ, s *xmpp.Session) *Iter {
	iq.Query.Item = nil
	iq.Type = stanza.GetIQ
	iter, start, err := s.IterIQ(ctx, iq.TokenReader())
	if err != nil {
		return &Iter{err: err}
	}
	var ver string
	for _, attr := range start.Attr {
		if attr.Name.Local == "ver" {
			ver = attr.Value
			break
		}
	}
	if ver == "" {
		ver = iq.Query.Ver
	}

	// Return the iterator which will parse the rest of the payload incrementally.
	return &Iter{
		iter: iter,
		ver:  ver,
	}
}

// IQ represents a user roster request or response.
// The zero value is a valid query for the roster.
type IQ struct {
	stanza.IQ

	Query struct {
		Ver  string `xml:"ver,attr"`
		Item []Item `xml:"item"`
	} `xml:"jabber:iq:roster query"`
}

type itemMarshaler struct {
	items []Item
	cur   xml.TokenReader
}

func (m *itemMarshaler) Token() (xml.Token, error) {
	if len(m.items) == 0 && m.cur == nil {
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
	return iq.IQ.Wrap(iq.payload())
}

// Payload returns a stream of XML tokekns that match the roster query payload
// without the IQ wrapper.
func (iq IQ) payload() xml.TokenReader {
	attrs := []xml.Attr{{Name: xml.Name{Local: "ver"}, Value: iq.Query.Ver}}

	return xmlstream.Wrap(
		&itemMarshaler{items: iq.Query.Item[:]},
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
	JID          jid.JID  `xml:"jid,attr,omitempty"`
	Name         string   `xml:"name,attr,omitempty"`
	Subscription string   `xml:"subscription,attr,omitempty"`
	Group        []string `xml:"group,omitempty"`
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (item Item) TokenReader() xml.TokenReader {
	var group []xml.TokenReader
	for _, g := range item.Group {
		group = append(group, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(g)),
			xml.StartElement{
				Name: xml.Name{Local: "group"},
			},
		))
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
		xmlstream.MultiReader(group...),
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

// Set creates a new roster item or updates an existing item.
func Set(ctx context.Context, s *xmpp.Session, item Item) error {
	q := IQ{
		IQ: stanza.IQ{Type: stanza.SetIQ},
	}
	q.Query.Item = append(q.Query.Item, item)
	resp, err := s.SendIQ(ctx, q.TokenReader())
	if err != nil {
		return err
	}
	return resp.Close()
}

// Delete removes a roster item from the users roster.
func Delete(ctx context.Context, s *xmpp.Session, j jid.JID) error {
	q := IQ{
		IQ: stanza.IQ{Type: stanza.SetIQ},
	}
	q.Query.Item = append(q.Query.Item, Item{
		JID:          j,
		Subscription: "remove",
	})
	resp, err := s.SendIQ(ctx, q.TokenReader())
	if err != nil {
		return err
	}
	return resp.Close()
}
