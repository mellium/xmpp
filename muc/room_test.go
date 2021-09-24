// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc_test

import (
	"context"
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
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
	m := mux.New(ns.Client, muc.HandleClient(h))
	server := mux.New(ns.Client,
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
			ns.Client,
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

var affiliationTestCases = []struct {
	Affiliation muc.Affiliation `xml:"affiliation,attr"`
	JID         jid.JID         `xml:"jid,attr"`
	Nick        string          `xml:"nick,attr"`
	Reason      string          `xml:"reason"`
	x           string
}{
	0: {
		x: `<item xmlns="http://jabber.org/protocol/muc#admin" affiliation="none" jid=""></item>`,
	},
	1: {
		Affiliation: muc.AffiliationMember,
		Nick:        "nick",
		JID:         jid.MustParse("me@example.net/removethis"),
		Reason:      "reason",
		x:           `<item xmlns="http://jabber.org/protocol/muc#admin" affiliation="member" jid="me@example.net" nick="nick"><reason xmlns="http://jabber.org/protocol/muc#admin">reason</reason></item>`,
	},
	2: {
		Affiliation: muc.AffiliationMember,
		Nick:        "nick",
		JID:         jid.MustParse("me@example.net/removethis"),
		x:           `<item xmlns="http://jabber.org/protocol/muc#admin" affiliation="member" jid="me@example.net" nick="nick"></item>`,
	},
}

func TestSetAffiliation(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	handled := make(chan string, 1)
	m := mux.New(ns.Client, muc.HandleClient(h))
	server := mux.New(
		ns.Client,
		mux.PresenceFunc("", xml.Name{Local: "x"}, func(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
			// Send back a self presence, indicating that the join is complete.
			p.To, p.From = p.From, p.To
			_, err := xmlstream.Copy(r, p.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: muc.NSUser, Local: "x"}},
			)))
			return err
		}),
		mux.IQFunc(stanza.SetIQ, xml.Name{Space: muc.NSAdmin, Local: "query"}, func(iq stanza.IQ, r xmlstream.TokenReadEncoder, _ *xml.StartElement) error {
			var buf strings.Builder
			defer func() {
				handled <- buf.String()
			}()
			e := xml.NewEncoder(&buf)
			_, err := xmlstream.Copy(e, xmlstream.Inner(r))
			if err != nil {
				return err
			}
			err = e.Flush()
			if err != nil {
				return err
			}
			_, err = xmlstream.Copy(r, iq.Result(nil))
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

	for i, tc := range affiliationTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err = channel.SetAffiliation(context.Background(), tc.Affiliation, tc.JID, tc.Nick, tc.Reason)
			if err != nil {
				t.Fatalf("error setting affiliation: %v", err)
			}
			x := <-handled
			if x != tc.x {
				t.Fatalf("wrong output:\nwant=%s,\n got=%s", tc.x, x)
			}
		})
	}
}

func TestSetSubject(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	handled := make(chan string, 1)
	m := mux.New(ns.Client, muc.HandleClient(h))
	server := mux.New(
		ns.Client,
		mux.PresenceFunc("", xml.Name{Local: "x"}, func(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
			// Send back a self presence, indicating that the join is complete.
			p.To, p.From = p.From, p.To
			_, err := xmlstream.Copy(r, p.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: muc.NSUser, Local: "x"}},
			)))
			return err
		}),
		mux.MessageFunc(stanza.GroupChatMessage, xml.Name{Local: "subject"}, func(_ stanza.Message, r xmlstream.TokenReadEncoder) error {
			d := xml.NewTokenDecoder(r)
			// Throw away the <message> token.
			_, err := d.Token()
			if err != nil {
				return nil
			}
			s := struct {
				XMLName xml.Name `xml:"subject"`
				Subject string   `xml:",chardata"`
			}{}
			err = d.Decode(&s)
			if err != nil {
				return err
			}
			handled <- s.Subject
			return nil
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

	const expected = "test"
	err = channel.Subject(context.Background(), expected)
	if err != nil {
		t.Fatalf("error setting subject: %v", err)
	}
	x := <-handled
	if x != expected {
		t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, x)
	}
}
