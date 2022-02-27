// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package bookmarks_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
)

func TestIntegrationFetch(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationFetch)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationFetch)
}

func integrationFetch(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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
	err = bookmarks.Publish(ctx, session, bookmarks.Channel{
		JID:      j,
		Name:     "The Bookmark",
		Autojoin: true,
	})
	if err != nil {
		t.Fatalf("error publishing bookmark: %v", err)
	}
	iter := bookmarks.Fetch(ctx, session)
	hasNext := iter.Next()
	if !hasNext {
		t.Fatalf("did not find any bookmarks")
	}

	bookmark := iter.Bookmark()
	if !bookmark.JID.Equal(j) {
		t.Fatalf("wrong JID: want=example.net, got=%v", bookmark.JID)
	}
	if bookmark.Name != "The Bookmark" {
		t.Fatalf("wrong name: want=The Bookmark, got=%q", bookmark.Name)
	}
	if !bookmark.Autojoin {
		t.Fatalf("expected autojoin to be set")
	}
	hasNext = iter.Next()
	if hasNext {
		t.Fatalf("too many bookmarks")
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iterator: %v", err)
	}

	err = bookmarks.Delete(ctx, session, j)
	if err != nil {
		t.Fatalf("error deleting bookmark: %v", err)
	}

	iter = bookmarks.Fetch(ctx, session)
	if iter.Next() {
		t.Fatalf("did not expect there to be any bookmarks!")
	}
	err = iter.Err()
	if err != nil {
		t.Fatalf("bookmark iteration with no bookmarks errored: %v", err)
	}
}
