// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package history_test

import (
	"context"
	"crypto/tls"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestIntegrationFetch(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.ListenC2S(),
		prosody.Modules("mam"),
	)
	prosodyRun(integrationFetch)
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
	var messages int
	messageHandler := history.NewHandler(mux.MessageHandlerFunc(func(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
		messages++
		return nil
	}))
	closed := make(chan struct{})
	t.Cleanup(func() {
		err = session.Close()
		if err != nil {
			t.Fatalf("error closing session: %v", err)
		}
		<-closed
	})
	go func() {
		m := mux.New(stanza.NSClient, history.Handle(messageHandler))
		err := session.Serve(m)
		if err != nil {
			t.Errorf("error from serve: %v", err)
		}
		close(closed)
	}()

	// Fetch history and make sure it's empty to start.
	res, err := history.Fetch(ctx, history.Query{}, j, session)
	if err != nil {
		t.Fatalf("error fetching history: %v", err)
	}
	if messages > 0 {
		t.Fatalf("%d messages returned but there should be nothing in history", messages)
	}
	if !res.Complete {
		t.Fatalf("expected empty result set to be complete")
	}

	// Send a message and then fetch history again to make sure that we get the
	// message back.
	err = session.Encode(ctx, struct {
		stanza.Message
		Body string `xml:"body"`
	}{
		Message: stanza.Message{
			Type: stanza.ChatMessage,
			To:   j,
		},
		Body: "test",
	})
	if err != nil {
		t.Fatalf("error sending message: %v", err)
	}
	res, err = history.Fetch(ctx, history.Query{}, j, session)
	if err != nil {
		t.Fatalf("error fetching history again: %v", err)
	}
	// We get the outbound and inbound version since we sent it to ourself.
	if messages != 2 {
		t.Fatalf("wrong number of messages returned: want=2, got=%d", messages)
	}
	if !res.Complete {
		t.Fatalf("expected result set to be complete")
	}
}
