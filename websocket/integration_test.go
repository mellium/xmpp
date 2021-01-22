// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package websocket_test

import (
	"context"
	"crypto/tls"
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
		integration.Cert("localhost:5281"),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationDialWebsocket)
}

func integrationDialWebsocket(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	d := websocket.Dialer{
		Origin: "http://localhost:5281/",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	conn, err := d.DialDirect(context.Background(), "wss://localhost:5281/xmpp-websocket")
	if err != nil {
		t.Fatalf("error dialing WebSocket connection: %v", err)
	}
	session, err := xmpp.NegotiateSession(
		context.TODO(), j.Domain(), j, conn, false,
		xmpp.NewNegotiator(xmpp.StreamConfig{
			WebSocket: true,
			Secure:    true,
			Features: []xmpp.StreamFeature{
				xmpp.SASL("", pass, sasl.Plain),
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
