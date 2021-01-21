// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package websocket_test

import (
	"context"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/websocket"
)

func TestIntegrationDialWebSocket(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.Cert("localhost"),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationDialWebsocket)
}

func integrationDialWebsocket(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	conn, err := websocket.DialDirect(context.Background(), "http://localhost:5280/", "ws://localhost:5280/xmpp-websocket")
	if err != nil {
		t.Fatalf("error dialing WebSocket connection: %v", err)
	}
	saslFeature := xmpp.SASL("", pass, sasl.Plain)
	saslFeature.Necessary &^= xmpp.Secure
	session, err := xmpp.NegotiateSession(
		context.TODO(), j.Domain(), j, conn, false,
		xmpp.NewNegotiator(xmpp.StreamConfig{
			WebSocket: true,
			Features: []xmpp.StreamFeature{
				saslFeature,
				xmpp.BindResource(),
			},
		}),
	)
	if err != nil {
		t.Fatalf("error negotiating session: %v", err)
	}
	go func() {
		err := session.Serve(nil)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()
	err = ping.Send(ctx, session, session.RemoteAddr())
	if err != nil {
		t.Errorf("error pinging: %v", err)
	}
}
