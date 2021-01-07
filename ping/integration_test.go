// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package ping_test

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"testing"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/internal/integration/sendxmpp"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

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
		m := mux.New(mux.IQFunc(stanza.GetIQ, xml.Name{Local: "ping", Space: ping.NS},
			func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
				close(gotPing)
				ping.Handler{}.HandleIQ(iq, t, start)
				return nil
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
		sendxmpp.Send(cmd, fmt.Sprintf(`<iq to="%s" id="123" type="get"><ping xmlns='urn:xmpp:ping'/></iq>`, session.LocalAddr()))
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
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
