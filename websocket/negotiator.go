// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package websocket

import (
	"context"
	"io"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/wskey"
	"mellium.im/xmpp/stream"
)

// Negotiator is like xmpp.NewNegotiator except that it uses the websocket
// subprotocol.
func Negotiator(cfg func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig) xmpp.Negotiator {
	xmppNegotiator := xmpp.NewNegotiator(cfg)
	return func(ctx context.Context, in, out *stream.Info, session *xmpp.Session, data interface{}) (xmpp.SessionState, io.ReadWriter, interface{}, error) {
		ctx = context.WithValue(ctx, wskey.Key{}, struct{}{})
		return xmppNegotiator(ctx, in, out, session, data)
	}
}
