// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package component_test

import (
	"context"
	"testing"

	"mellium.im/xmpp/component"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
)

const (
	domain = "component.localhost"
	secret = "fo0b4r"
)

func TestIntegrationComponentClient(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
		prosody.Component(domain, secret),
	)
	prosodyRun(integrationComponentClient)
}

func integrationComponentClient(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j := jid.MustParse(domain)
	conn, err := cmd.ComponentConn(ctx)
	if err != nil {
		t.Errorf("error dialing connection: %v", err)
	}
	_, err = component.NewSession(context.Background(), j, []byte(secret), conn, false)
	if err != nil {
		t.Errorf("error negotiating session: %v", err)
	}
}
