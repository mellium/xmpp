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
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
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
		m := mux.New(muc.HandleClient(mucClient))
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
		m := mux.New(muc.HandleClient(inviteClient))
		// TODO: implement server side of invites and test the handler here.
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
