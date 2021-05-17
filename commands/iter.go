// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands

import (
	"context"

	"mellium.im/xmpp"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// Iter is an iterator over Command's.
type Iter struct {
	*disco.ItemIter
}

// Command returns the last command parsed by the iterator.
func (i Iter) Command() Command {
	item := i.Item()
	return Command{
		JID:  item.JID,
		Name: item.Name,
		Node: item.Node,
	}
}

// Fetch requests a list of commands.
//
// The iterator must be closed before anything else is done on the session or it
// will become invalid.
// Any errors encountered while creating the iter are deferred until the iter is
// used.
func Fetch(ctx context.Context, to jid.JID, s *xmpp.Session) Iter {
	return FetchIQ(ctx, stanza.IQ{To: to}, s)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) Iter {
	return Iter{ItemIter: disco.FetchItemsIQ(ctx, NS, iq, s)}
}
