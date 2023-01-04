// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package blocklist_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/blocklist"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
)

func TestIntegrationBlock(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
		prosody.Modules("blocklist"),
	)
	prosodyRun(integrationBlock)
}

func integrationBlock(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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

	// Fetch the block list and make sure it's empty.
	iter := blocklist.Fetch(ctx, session)
	if iter.Next() {
		t.Fatalf("blocklist already contains items")
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iter: %v", err)
	}

	// Add an item to the block list, then fetch it again and make sure we get the
	// item back.
	var (
		a = jid.MustParse("a@example.net")
		b = jid.MustParse("b@example.net")
	)
	err = blocklist.Add(ctx, session, a, b)
	if err != nil {
		t.Fatalf("error adding JIDs to the block list: %v", err)
	}

	var jids []jid.JID
	iter = blocklist.Fetch(ctx, session)
	for iter.Next() {
		jids = append(jids, iter.JID())
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing iter: %v", err)
	}
	if len(jids) != 2 {
		t.Fatalf("got different number of JIDs than expected: want=%d, got=%d", 2, len(jids))
	}
}
