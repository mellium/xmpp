// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks

import (
	"context"
	"encoding/xml"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/pubsub"
	"mellium.im/xmpp/stanza"
)

// Fetch requests all bookmarks from the server and returns an iterator over the
// results (blocking until the response is received and the iterator is fully
// consumed or closed).
func Fetch(ctx context.Context, s *xmpp.Session) *Iter {
	return FetchIQ(ctx, stanza.IQ{}, s)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) *Iter {
	iq.Type = stanza.GetIQ
	iter := pubsub.FetchIQ(ctx, iq, s, pubsub.Query{
		Node: NS,
	})
	return &Iter{
		iter: iter,
	}
}

// Iter is an iterator over bookmarks.
type Iter struct {
	iter    *pubsub.Iter
	current Channel
	err     error
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	id, r := i.iter.Item()
	var bookmark Channel
	i.err = xml.NewTokenDecoder(r).Decode(&bookmark)
	if i.err != nil {
		return false
	}
	j, err := jid.Parse(id)
	if err != nil {
		return false
	}
	i.current = bookmark
	i.current.JID = j
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Bookmark returns the last bookmark parsed by the iterator.
func (i *Iter) Bookmark() Channel {
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
