// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"context"
	"encoding/xml"
	"errors"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/paging"
	"mellium.im/xmpp/stanza"
)

const (
	defPageSize = 32
)

// ItemsQuery is the payload of a query for a node's items.
type ItemsQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#items query"`
	Node    string   `xml:"node,attr,omitempty"`
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

// ItemIter is an iterator over discovered items.
// It supports paging
type ItemIter struct {
	iter    *paging.Iter
	current items.Item
	err     error
	ctx     context.Context
	session *xmpp.Session
}

// Next returns true if there are more items to decode.
func (i *ItemIter) Next() bool {
	if i.err != nil {
		return false
	}
	next := i.iter.Next()
	nextPage := i.iter.NextPage()
	if !next && nextPage == nil {
		return false
	}
	// If there is a next element and we don't need to turn the page, just decode
	// is like normal.
	if next {
		start, r := i.iter.Current()
		// If we encounter a lone token that doesn't begin with a start element (eg.
		// a comment) skip it. This should never happen with XMPP, but we don't want
		// to panic in case this somehow happens so just skip it.
		if start == nil {
			return i.Next()
		}
		d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), r))
		item := items.Item{}
		i.err = d.Decode(&item)
		if i.err != nil {
			return false
		}
		i.current = item
		return true
	}

	// Turn the page.
	i.err = i.iter.Close()
	if i.err != nil {
		return false
	}
	// TODO: set context based on a deadline?
	i.iter = FetchItems(i.ctx, i.current, i.session).iter
	return i.Next()
}

// Err returns the last error encountered by the iterator (if any).
func (i *ItemIter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Item returns the last item parsed by the iterator.
func (i *ItemIter) Item() items.Item {
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

// FetchItems discovers a set of items associated with a JID and optional node of
// the provided item.
// The Name attribute of the query item is ignored.
// An empty Node means to query the root items for the JID.
// It blocks until a response is received.
//
// The iterator must be closed before anything else is done on the session.
// Any errors encountered while creating the iter are deferred until the iter is
// used.
func FetchItems(ctx context.Context, item items.Item, s *xmpp.Session) *ItemIter {
	return FetchItemsIQ(ctx, item.Node, stanza.IQ{To: item.JID}, s)
}

// FetchItemsIQ is like FetchItems but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchItemsIQ(ctx context.Context, node string, iq stanza.IQ, s *xmpp.Session) *ItemIter {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	query := ItemsQuery{
		Node: node,
	}
	iter, _, err := s.IterIQ(ctx, iq.Wrap(query.TokenReader()))
	if err != nil {
		return &ItemIter{err: err}
	}
	return &ItemIter{iter: paging.WrapIter(iter, defPageSize), ctx: ctx, session: s}
}

// ErrSkipItem is used as a return value from WalkItemFuncs to indicate that the
// node named in the call is to be skipped.
// It is not returned as an error by any function.
var ErrSkipItem = errors.New("skip this item")

// WalkItemFunc is the type of function called by WalkItem to visit each item in
// an item hierarchy.
// Item nodes are unique and absolute (in particular they should not be treated
// like paths, even if a particular implementation uses paths for node names).
//
// The error result returned by the function controls how WalkItem continues.
// If the function returns the special value ErrSkipItem, WalkItem skips the
// current item.
// Otherwise, if the function returns a non-nil error, WalkItem stops entirely
// and returns that error.
//
// The error reports an error related to the item, signaling that WalkItem will
// not walk into that item.
// The function may decide how to handle that error, including returning it to
// stop walking the entire tree.
//
// The function is called before querying for an item to allow SkipItem to
// bypass the query entirely.
// If an error occurs while making the query, the function will be called again
// with the same item to report the error.
type WalkItemFunc func(level int, item items.Item, err error) error

// WalkItem walks the tree rooted at the JID, calling fn for each item in the
// tree, including root.
// To query the root, leave item.Node empty.
// The Name attribute of the query item is ignored.
//
// All errors that arise visiting items are filtered by fn: see the WalkItemFunc
// documentation for details.
//
// The items are walked in wire order which may make the output
// non-deterministic.
func WalkItem(ctx context.Context, item items.Item, s *xmpp.Session, fn WalkItemFunc) error {
	return walkItem(ctx, 0, 0, []items.Item{item}, s, fn)
}

func ignoredErr(err error) bool {
	return errors.Is(err, stanza.Error{Condition: stanza.FeatureNotImplemented}) || errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable})
}

func walkItem(ctx context.Context, level, itemIdx int, items []items.Item, s *xmpp.Session, fn WalkItemFunc) error {
	last := len(items) - 1
	item := items[itemIdx]
	err := fn(level, item, nil)
	if err != nil {
		if err == ErrSkipItem {
			err = nil
		}
		return err
	}

	// Look for loops and duplicates.
	for n, oldItem := range items {
		if n == itemIdx {
			continue
		}
		if oldItem.Node == item.Node && oldItem.JID.Equal(item.JID) {
			return nil
		}
	}

	items, err = appendItems(ctx, s, itemIdx, items)
	if ignoredErr(err) {
		err = nil
	}
	if err != nil {
		// Report the error with a second call to fn.
		err = fn(level, item, err)
		if err != nil {
			return err
		}
	}

	for n := range items[last+1:] {
		err = walkItem(ctx, level+1, n+last+1, items, s, fn)
		if err != nil {
			if err == ErrSkipItem {
				continue
			}
			return err
		}
	}
	return nil
}

func appendItems(ctx context.Context, s *xmpp.Session, itemIdx int, items []items.Item) (i []items.Item, err error) {
	iter := FetchItems(ctx, items[itemIdx], s)
	defer func() {
		e := iter.Close()
		if err == nil {
			err = e
		}
	}()
	for iter.Next() {
		items = append(items, iter.Item())
	}
	return items, iter.Err()
}
