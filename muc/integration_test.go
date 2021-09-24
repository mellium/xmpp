// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package muc_test

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestIntegrationJoinRoom(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.MUC("muc.localhost"),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationJoinRoom)
}

func integrationJoinRoom(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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
	mucClient := &muc.Client{}
	go func() {
		m := mux.New(ns.Client, muc.HandleClient(mucClient))
		err := session.Serve(m)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()

	// Fetch rooms and make sure they're empty.
	roomJID := jid.MustParse("bridgecrew@muc.localhost/Picard")
	iter := disco.FetchItems(ctx, items.Item{
		JID: roomJID.Domain(),
	}, session)
	for iter.Next() {
		t.Errorf("did not expect any rooms initially, got: %v", iter.Item())
	}
	if err = iter.Err(); err != nil {
		t.Fatalf("error fetching rooms: %v", err)
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iter: %v", err)
	}

	channel, err := mucClient.Join(ctx, roomJID, session)
	if err != nil {
		t.Fatalf("error joining MUC: %v", err)
	}

	iter = disco.FetchItems(ctx, items.Item{
		JID: roomJID.Domain(),
	}, session)
	for iter.Next() {
		t.Errorf("did not expect any private rooms, got: %v", iter.Item())
	}
	if err = iter.Err(); err != nil {
		t.Fatalf("error fetching rooms: %v", err)
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iter: %v", err)
	}

	roomForm, err := muc.GetConfig(ctx, roomJID.Bare(), session)
	if err != nil {
		t.Fatalf("error fetching config: %v", err)
	}
	_, err = roomForm.Set("muc#roomconfig_publicroom", true)
	if err != nil {
		t.Errorf("error making room public: %v", err)
	}

	err = muc.SetConfig(ctx, roomJID.Bare(), roomForm, session)
	if err != nil {
		t.Fatalf("error setting room config: %v", err)
	}

	// Fetch rooms again and make sure the new one was created.
	var list []items.Item
	iter = disco.FetchItems(ctx, items.Item{
		JID: roomJID.Domain(),
	}, session)
	for iter.Next() {
		list = append(list, iter.Item())
	}
	if err = iter.Err(); err != nil {
		t.Fatalf("error fetching rooms: %v", err)
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing final iter: %v", err)
	}
	if len(list) != 1 || !list[0].JID.Equal(roomJID.Bare()) {
		t.Fatalf("wrong rooms created: want=%v, got=%v", roomJID.Bare(), list)
	}

	err = channel.Leave(ctx, "")
	if err != nil {
		t.Fatalf("error leaving room: %v", err)
	}

	// Fetch rooms and make sure they're empty (room was not persistent and was
	// destroyed when we left, indicating that we did in fact leave correctly).
	iter = disco.FetchItems(ctx, items.Item{
		JID: roomJID.Domain(),
	}, session)
	for iter.Next() {
		t.Errorf("did not expect any rooms after part, got: %v", iter.Item())
	}
	if err = iter.Err(); err != nil {
		t.Fatalf("error fetching rooms: %v", err)
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iter: %v", err)
	}
}

func TestIntegrationJoinErr(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.MUC("muc.localhost"),
		prosody.Channel("muc.localhost", prosody.ChannelConfig{
			Localpart:  "bridgecrew",
			Admins:     []string{"me@localhost"},
			Name:       "Bridge Crew",
			Persistent: true,
			Public:     true,
			Pass:       "nojoin",
		}),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationJoinErr)
}

func integrationJoinErr(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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
	mucClient := &muc.Client{}
	go func() {
		m := mux.New(ns.Client, muc.HandleClient(mucClient))
		err := session.Serve(m)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()

	roomJID := jid.MustParse("bridgecrew@muc.localhost/Picard")
	_, err = mucClient.Join(ctx, roomJID, session)
	noAuth := stanza.Error{
		Condition: stanza.NotAuthorized,
	}
	if !errors.Is(err, noAuth) {
		t.Fatalf("wrong error type, want=%T (%[1]v), got=%T (%[2]v)", noAuth, err)
	}
}
