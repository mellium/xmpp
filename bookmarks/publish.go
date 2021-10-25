// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks

import (
	"context"

	"mellium.im/xmpp"
	"mellium.im/xmpp/pubsub"
	"mellium.im/xmpp/stanza"
)

// Publish creates or updates the bookmark.
func Publish(ctx context.Context, s *xmpp.Session, b Channel) error {
	return PublishIQ(ctx, s, stanza.IQ{}, b)
}

// PublishIQ is like Publish except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func PublishIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, b Channel) error {
	iq.Type = stanza.SetIQ
	_, err := pubsub.PublishIQ(ctx, s, iq, NS, b.JID.String(), b.TokenReader())
	return err
}
