// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package s2s_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/s2s"
)

func TestIntegrationS2S(t *testing.T) {
	const hostname = "origin"
	origin := jid.MustParse(hostname)

	runProsody := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.ListenS2S(),
		prosody.TrustAll(),
		prosody.Bidi(),
		integration.ClientCert(hostname),
	)
	runProsody(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		j, _ := cmd.User()
		session, err := cmd.DialServer(ctx, j.Domain(), origin, t,
			xmpp.StartTLS(&tls.Config{
				InsecureSkipVerify:   true,
				GetClientCertificate: cmd.ClientCert,
			}),
			s2s.Bidi(),
			xmpp.SASL("", "", s2s.TLSAuth()),
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
	})
}
