// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package ping_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/ping"
)

func TestIntegrationSendPing(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationSendPing)
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
	err = ping.Send(context.TODO(), session, session.RemoteAddr())
	if err != nil {
		t.Errorf("error pinging: %v", err)
	}
}
