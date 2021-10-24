// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package pubsub_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/pubsub"
)

func TestIntegrationConfig(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationConfig)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationConfig)
}

func integrationConfig(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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

	data, err := pubsub.GetDefaultConfig(ctx, session)
	if err != nil {
		t.Fatalf("error getting default config: %v", err)
	}
	const (
		field      = "pubsub#max_items"
		defVal     = "1"
		initialVal = "3"
		updateVal  = "2"
	)
	s, _ := data.GetString(field)
	if s != defVal {
		t.Fatalf("wrong value from def config: want=%q, got=%q", defVal, s)
	}
	ok, err := data.Set(`pubsub#max_items`, initialVal)
	if err != nil {
		t.Fatalf("error setting initial %s: %v", field, err)
	}
	err = pubsub.CreateNode(ctx, session, t.Name(), data)
	if err != nil {
		t.Fatalf("error creating node: %v", err)
	}
	data, err = pubsub.GetConfig(ctx, session, t.Name())
	if err != nil {
		t.Fatalf("error getting initial config: %v", err)
	}
	s, _ = data.GetString(field)
	if s != initialVal {
		t.Fatalf("wrong value from new config: want=%q, got=%q", initialVal, s)
	}
	ok, err = data.Set(field, updateVal)
	if err != nil {
		t.Fatalf("error setting %s: %v", field, err)
	}
	if !ok {
		t.Fatalf("no field value %s found", field)
	}

	err = pubsub.SetConfig(ctx, session, t.Name(), data)
	if err != nil {
		t.Fatalf("error setting new config: %v", err)
	}

	data, err = pubsub.GetConfig(ctx, session, t.Name())
	if err != nil {
		t.Fatalf("error getting new config: %v", err)
	}
	s, _ = data.GetString(field)
	if s != updateVal {
		t.Fatalf("wrong value from config: want=%q, got=%q", updateVal, s)
	}
}
