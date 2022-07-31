// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package xtime_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/jackal"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/xtime"
)

func TestIntegrationRequestTime(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationRequestTime)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationRequestTime)

	jackalRun := jackal.Test(context.TODO(), t,
		integration.Log(),
		jackal.ListenC2S(),
	)
	jackalRun(integrationRequestTime)
}

func integrationRequestTime(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	session, err := cmd.DialClient(ctx, j, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", pass, sasl.Plain, sasl.ScramSha256),
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
	tt, err := xtime.Get(ctx, session, session.RemoteAddr())
	if err != nil {
		t.Errorf("error getting time: %v", err)
	}
	if now := time.Now().UTC().Format(time.RFC3339); tt.UTC().Format(time.RFC3339) != now {
		t.Errorf("wrong time: want=%v, got=%v", now, tt)
	}
}
