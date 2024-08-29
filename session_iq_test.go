// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

func TestResponseToTimedOutIQ(t *testing.T) {
	// Regression test for #399

	ctx, cancel := context.WithCancel(context.Background())
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(toks xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// Cancel the context after the server has started processing the
			// response.
			// Technically I think this may be flakey, but I couldn't think of another
			// way to reproduce the problem and, at least for now, the way the
			// buffering in the ClientServer works it should all be fine, but small
			// changes could break this later.
			cancel()
			iq, err := stanza.NewIQ(*start)
			if err != nil {
				return err
			}
			return ping.Handler{}.HandleIQ(iq, toks, start)
		}),
		xmpptest.ClientHandlerFunc(func(toks xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(xmlstream.Discard(), toks)
			return err
		}),
	)
	/* #nosec */
	defer func() {
		err := cs.Close()
		if err != nil {
			t.Fatalf("error closing client/server: %v", err)
		}
	}()

	_, err := cs.Client.EncodeIQ(ctx, ping.IQ{
		IQ: stanza.IQ{
			Type: stanza.GetIQ,
		},
	})
	if err != context.Canceled {
		t.Fatalf("error encoding IQ: %v", err)
	}
}
