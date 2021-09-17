// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package websocket_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/websocket"
)

func TestIntegrationDialWebSocket(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.WebSocket(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationDialWebsocket)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.WebSocket(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationDialWebsocketUnix)
}

func integrationDialWebsocketUnix(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	conn, err := cmd.HTTPConn(ctx, true)
	if err != nil {
		t.Fatalf("error getting HTTPS connection: %v", err)
	}
	session, err := websocket.NewClient(ctx, "wss://localhost/xmpp", "wss://localhost/xmpp", j, conn, xmpp.SASL("", pass, sasl.Plain), xmpp.BindResource())
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

func integrationDialWebsocket(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	d := websocket.Dialer{
		Origin: "http://localhost:" + cmd.HTTPSPort() + "/",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	conn, err := d.DialDirect(context.Background(), "wss://localhost:"+cmd.HTTPSPort()+"/xmpp-websocket")
	if err != nil {
		t.Fatalf("error dialing WebSocket connection: %v", err)
	}
	session, err := xmpp.NewSession(
		context.TODO(), j.Domain(), j, conn,
		xmpp.Secure,
		websocket.Negotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
			return xmpp.StreamConfig{
				Features: []xmpp.StreamFeature{
					xmpp.SASL("", pass, sasl.Plain),
					xmpp.BindResource(),
				},
			}
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
