// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc_test

import (
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestMediatedInvite(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	inviteChan := make(chan muc.Invitation, 1)
	m := mux.New(muc.HandleClient(h))
	server := mux.New(
		mux.PresenceFunc("", xml.Name{Local: "x"}, func(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
			// Send back a self presence, indicating that the join is complete.
			p.To, p.From = p.From, p.To
			_, err := xmlstream.Copy(r, p.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: muc.NSUser, Local: "x"}},
			)))
			return err
		}),
		mux.MessageFunc(stanza.NormalMessage, xml.Name{Local: "x"}, func(m stanza.Message, r xmlstream.TokenReadEncoder) error {
			d := xml.NewTokenDecoder(r)
			_, err := d.Token()
			if err != nil {
				close(inviteChan)
				return err
			}
			var invite muc.Invitation
			err = d.Decode(&invite)
			inviteChan <- invite
			return err
		}),
	)
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(m),
		xmpptest.ServerHandler(server),
	)

	channel, err := h.Join(context.Background(), j, s.Client)
	if err != nil {
		t.Fatalf("error joining: %v", err)
	}

	const expectedReason = "reason"
	err = channel.Invite(context.Background(), expectedReason, s.Client.LocalAddr())
	if err != nil {
		t.Fatalf("error sending invite: %v", err)
	}
	invite := <-inviteChan

	if invite.Reason != expectedReason {
		t.Errorf("wrong reason: want=%v, got=%v", expectedReason, invite.Reason)
	}
}

func TestDirectInvite(t *testing.T) {
	inviteChan := make(chan muc.Invitation, 1)
	s := xmpptest.NewClientServer(
		xmpptest.ServerHandler(mux.New(
			muc.HandleInvite(func(invite muc.Invitation) {
				inviteChan <- invite
			}),
		)),
	)

	const expectedReason = "reason"
	muc.Invite(context.Background(), s.Server.LocalAddr(), muc.Invitation{
		Reason: expectedReason,
	}, s.Client)
	invite := <-inviteChan

	if invite.Reason != expectedReason {
		t.Errorf("wrong reason: want=%v, got=%v", expectedReason, invite.Reason)
	}
}
