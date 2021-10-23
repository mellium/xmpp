// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package pubsub

import (
	"context"
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/paging"
	"mellium.im/xmpp/stanza"
)

// Query represents the options for fetching and iterating over pubsub items.
type Query struct {
	// Node is the ID of a node to query.
	Node string

	// Item is a specific item to fetch by its ID.
	// Most users should use one of the methods specifically for fetching
	// individual items instead of filtering the results and using an iterator
	// over 0 or 1 items.
	Item string

	// MaxItems can be used to restrict results to the most recent items.
	MaxItems uint64
}

// Fetch requests all items in a node and returns an iterator over each item.
//
// Processing the session will become blocked until the iterator is closed.
// Any errors encountered while creating the iter are deferred until the iter is
// used.
func Fetch(ctx context.Context, s *xmpp.Session, q Query) *Iter {
	return FetchIQ(ctx, stanza.IQ{}, s, q)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, q Query) *Iter {
	iq.Type = stanza.GetIQ
	queryAttrs := []xml.Attr{{
		Name:  xml.Name{Local: "node"},
		Value: q.Node,
	}}
	if q.MaxItems > 0 {
		queryAttrs = append(queryAttrs, xml.Attr{
			Name:  xml.Name{Local: "max_items"},
			Value: strconv.FormatUint(q.MaxItems, 10),
		})
	}
	if q.Item != "" {
		queryAttrs = append(queryAttrs, xml.Attr{
			Name:  xml.Name{Local: "item"},
			Value: q.Item,
		})
	}
	iter, _, err := s.IterIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			nil,
			xml.StartElement{Name: xml.Name{Local: "items"}, Attr: queryAttrs},
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "pubsub"}},
	), iq)
	return &Iter{
		iter: paging.WrapIter(iter, 0),
		err:  err,
	}
}

// Iter is an iterator over payload items.
type Iter struct {
	iter    *paging.Iter
	current xml.TokenReader
	currID  string
	err     error
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
	i.currID = ""
	i.current = r
	for _, attr := range start.Attr {
		if attr.Name.Local == "id" {
			i.currID = attr.Value
			break
		}
	}
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Item returns the last item parsed by the iterator.
// If no payloads were requested in the original query the reader may be nil.
func (i *Iter) Item() (id string, r xml.TokenReader) {
	return i.currID, i.current
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
