// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package ping_test

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/xml"
	"testing"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/aioxmpp"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/mcabber"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/internal/integration/sendxmpp"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

//go:embed aioxmpp_integration_test.py
var pingScript string

func TestIntegrationSendPing(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationSendPing)
	prosodyRun(integrationRecvPing)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationSendPing)
}

func integrationRecvPing(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	gotPing := make(chan struct{})
	p := cmd.C2SPort()
	j, pass := cmd.User()
	session, err := cmd.DialClient(ctx, j, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", pass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting: %v", err)
	}
	go func() {
		m := mux.New(stanza.NSClient, mux.IQFunc(stanza.GetIQ, xml.Name{Local: "ping", Space: ping.NS},
			func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
				err := ping.Handler{}.HandleIQ(iq, t, start)
				gotPing <- struct{}{}
				return err
			},
		))
		err := session.Serve(m)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()
	sendxmppRun := sendxmpp.Test(ctx, t,
		integration.Log(),
		sendxmpp.ConfigFile(sendxmpp.Config{
			JID:      j,
			Port:     p,
			Password: pass,
		}),
		sendxmpp.Raw(),
		sendxmpp.TLS(),
	)
	sendxmppRun(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		err := sendxmpp.Ping(cmd, session.LocalAddr())
		if err != nil {
			t.Fatalf("error sending ping: %v", err)
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-gotPing:
		}
	})
	mcabberRun := mcabber.Test(ctx, t,
		mcabber.ConfigFile(mcabber.Config{
			JID:      j,
			Password: pass,
			Port:     p,
		}),
	)
	mcabberRun(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		err := mcabber.Ping(cmd, session.LocalAddr())
		if err != nil {
			t.Fatalf("error sending ping: %v", err)
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case err := <-cmd.Done():
			if err != nil {
				t.Errorf("command errored: %v", err)
			}
			t.Errorf("did not receive ping before command shutdown")
		case <-gotPing:
		}
	})
	aioxmppRun := aioxmpp.Test(ctx, t,
		integration.Log(),
		aioxmpp.ConfigFile(aioxmpp.Config{
			JID:      j,
			Password: pass,
			Port:     p,
		}),
		aioxmpp.Import("Ping", pingScript),
		aioxmpp.Args("-j", session.LocalAddr().String()),
	)
	aioxmppRun(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			t.Fatalf("context timed out: %v", ctx.Err())
		case err := <-cmd.Done():
			if err != nil {
				t.Errorf("command errored: %v", err)
			}
			t.Errorf("did not receive ping before command shutdown")
		case <-gotPing:
		}
	})
}

func integrationSendPing(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	session, err := cmd.DialClient(ctx, j, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", pass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting: %v", err)
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
