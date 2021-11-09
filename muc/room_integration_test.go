// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package muc_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const (
	userOne  = "foo@localhost"
	userTwo  = "bar@localhost"
	userPass = "Pass"
)

func TestIntegrationMediatedInvite(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.MUC("muc.localhost"),
		prosody.CreateUser(context.TODO(), userOne, userPass),
		prosody.CreateUser(context.TODO(), userTwo, userPass),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationMediatedInvite)
}

func TestIntegrationSetAffiliation(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.MUC("muc.localhost"),
		prosody.Channel("muc.localhost", prosody.ChannelConfig{
			Localpart:  "bridgecrew",
			Admins:     []string{userOne},
			Name:       "Bridge Crew",
			Persistent: true,
			Public:     true,
		}),
		prosody.CreateUser(context.TODO(), userOne, userPass),
		prosody.CreateUser(context.TODO(), userTwo, userPass),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationSetAffiliation)
}

func integrationMediatedInvite(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	userOneJID := jid.MustParse(userOne)
	userTwoJID := jid.MustParse(userTwo)
	userOneSession, err := cmd.DialClient(ctx, userOneJID, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", userPass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting %s: %v", userOne, err)
	}
	userTwoSession, err := cmd.DialClient(ctx, userTwoJID, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", userPass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting %s: %v", userTwo, err)
	}

	mucClient := &muc.Client{}
	go func() {
		m := mux.New(stanza.NSClient, muc.HandleClient(mucClient))
		err := userOneSession.Serve(m)
		if err != nil {
			t.Logf("error from %s serve: %v", userOne, err)
		}
	}()
	errChan := make(chan error)
	go func(errChan chan<- error) {
		inviteClient := &muc.Client{
			HandleInvite: func(i muc.Invitation) {
				errChan <- nil
			},
		}
		m := mux.New(stanza.NSClient, muc.HandleClient(inviteClient))
		err := userTwoSession.Serve(m)
		if err != nil {
			t.Logf("error from %s serve: %v", userTwo, err)
		}
	}(errChan)

	roomJID := jid.MustParse("bridgecrew@muc.localhost/Picard")
	channel, err := mucClient.Join(ctx, roomJID, userOneSession)
	if err != nil {
		t.Fatalf("error joining MUC: %v", err)
	}

	err = channel.Invite(ctx, "invited!", userTwoSession.LocalAddr())
	if err != nil {
		t.Fatalf("error sending invite: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("error receiving invite: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("invite not received: %v", ctx.Err())
	}
}

func integrationSetAffiliation(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	userOneJID := jid.MustParse(userOne)
	userTwoJID := jid.MustParse(userTwo)
	userOneSession, err := cmd.DialClient(ctx, userOneJID, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", userPass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting %s: %v", userOne, err)
	}
	userTwoSession, err := cmd.DialClient(ctx, userTwoJID, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", userPass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting %s: %v", userTwo, err)
	}

	mucClientOne := &muc.Client{}
	go func() {
		m := mux.New(stanza.NSClient, muc.HandleClient(mucClientOne))
		err := userOneSession.Serve(m)
		if err != nil {
			t.Logf("error from %s serve: %v", userOne, err)
		}
	}()

	itemChan := make(chan muc.Item)
	mucClientTwo := &muc.Client{
		HandleUserPresence: func(_ stanza.Presence, i muc.Item) {
			itemChan <- i
		},
	}
	go func(itemChan chan<- muc.Item) {
		m := mux.New(stanza.NSClient, muc.HandleClient(mucClientTwo))
		err := userTwoSession.Serve(m)
		if err != nil {
			t.Logf("error from %s serve: %v", userTwo, err)
		}
	}(itemChan)

	roomJID := jid.MustParse("bridgecrew@muc.localhost/Picard")
	channelOne, err := mucClientOne.Join(ctx, roomJID, userOneSession)
	if err != nil {
		t.Fatalf("error joining MUC as %s: %v", roomJID.Resourcepart(), err)
	}

	roomJIDTwo, err := roomJID.WithResource("CrusherMD")
	if err != nil {
		t.Fatalf("bad resource in test: %v", err)
	}
	_, err = mucClientTwo.Join(ctx, roomJIDTwo, userTwoSession)
	if err != nil {
		t.Fatalf("error joining MUC as %s: %v", roomJIDTwo.Resourcepart(), err)
	}

	err = channelOne.SetAffiliation(ctx, muc.AffiliationMember, userTwoJID, "Crusher", "Permission to speak freely")
	if err != nil {
		t.Fatalf("error setting affiliation: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	select {
	case item := <-itemChan:
		if item.Affiliation != muc.AffiliationMember {
			t.Fatalf("wrong affiliation: want=%v, got=%v", muc.AffiliationMember, item.Affiliation)
		}
	case <-ctx.Done():
		t.Fatalf("invite not received: %v", ctx.Err())
	}
}
