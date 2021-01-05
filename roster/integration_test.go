// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package roster_test

import (
	"context"
	"crypto/tls"
	"reflect"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
)

func TestIntegrationRoster(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationRoster)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationRoster)
}

func integrationRoster(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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

	// The initial roster should be empty.
	iter := roster.Fetch(ctx, session)
	for iter.Next() {
		t.Errorf("did not expect the initial roster to have any items, got: %v", iter.Item())
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("error iterating empty roster: %v", err)
	}
	if err := iter.Close(); err != nil {
		t.Fatalf("error closing empty roster iter: %v", err)
	}

	rosterItem := jid.MustParse("them@localhost")
	firstItem := roster.Item{
		JID:   rosterItem,
		Name:  "name",
		Group: []string{"group"},
	}
	err = roster.Set(ctx, session, firstItem)
	if err != nil {
		t.Errorf("error adding first JID to roster: %v", err)
	}

	// TODO: test that we get (and can handle) the roster push.

	// The roster should now have the item we added.
	iter = roster.Fetch(ctx, session)
	var foundItem bool
	for iter.Next() {
		if foundItem {
			t.Errorf("got unexpected item after first item: %v", iter)
		}
		item := iter.Item()
		// Prosody sets the subscription to "none" if we provide an empty string,
		// while ejabberd leaves it empty. Either is acceptable.
		if item.Subscription == "none" {
			item.Subscription = ""
		}
		if !reflect.DeepEqual(firstItem, item) {
			t.Errorf("first roster item was incorrect: want=%v, got=%v", firstItem, item)
		}
		foundItem = true
	}
	if !foundItem {
		t.Fatalf("expected item to have been added to the roster, but got nothing back")
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("error iterating empty roster: %v", err)
	}
	if err := iter.Close(); err != nil {
		t.Fatalf("error closing empty roster iter: %v", err)
	}

	err = roster.Delete(ctx, session, rosterItem)
	if err != nil {
		t.Errorf("error adding first JID to roster: %v", err)
	}

	// After deleting the item, the roster should be empty again.
	iter = roster.Fetch(ctx, session)
	for iter.Next() {
		t.Errorf("did not expect the final roster to have any items, got: %v", iter.Item())
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("error iterating final empty roster: %v", err)
	}
	if err := iter.Close(); err != nil {
		t.Fatalf("error closing final empty roster iter: %v", err)
	}
}
