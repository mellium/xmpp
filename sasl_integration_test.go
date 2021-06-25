// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package xmpp_test

import (
	"context"
	"testing"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/mcabber"
	"mellium.im/xmpp/internal/integration/mellium"
	"mellium.im/xmpp/jid"
)

func TestMain(m *testing.M) {
	mellium.TestMain(m)
}

func TestIntegrationSASLClient(t *testing.T) {
	const pass = "testpass"
	melliumRun := mellium.Test(context.TODO(), t,
		integration.Cert("localhost"),
		mellium.ConfigFile(mellium.Config{
			ListenC2S: true,
		}),
		integration.User(jid.MustParse("me@localhost"), pass),
	)
	melliumRun(integrationSASLClient)
}

func integrationSASLClient(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	p := cmd.C2SPort()
	mcabberRun := mcabber.Test(context.TODO(), t,
		mcabber.ConfigFile(mcabber.Config{
			JID:      j,
			Password: pass,
			Port:     p,
		}),
	)
	mcabberRun(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		t.Log("Connected successfully!")
	})
}
