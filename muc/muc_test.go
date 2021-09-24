// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc_test

import (
	"context"
	"encoding/xml"
	"errors"
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

func TestJoinPartMuc(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	m := mux.New(ns.Client, muc.HandleClient(h))
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(m),
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// Send back a self presence, indicating that the join is complete.
			p, err := stanza.NewPresence(*start)
			if err != nil {
				return err
			}
			p.To, p.From = p.From, p.To
			_, err = xmlstream.Copy(t, p.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: muc.NSUser, Local: "x"}},
			)))
			return err
		}),
	)

	channel, err := h.Join(context.Background(), j, s.Client)
	if err != nil {
		t.Fatalf("error joining: %v", err)
	}

	if !channel.Me().Equal(j) {
		t.Errorf("wrong JID: want=%v, got=%v", j, channel.Me())
	}
	if !channel.Addr().Equal(j.Bare()) {
		t.Errorf("wrong JID: want=%v, got=%v", j.Bare(), channel.Addr())
	}

	err = channel.Leave(context.Background(), "")
	if err != nil {
		t.Fatalf("error leaving: %v", err)
	}
	if channel.Joined() {
		t.Errorf("expected channel to be unjoined")
	}
}

func TestJoinError(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	m := mux.New(ns.Client, muc.HandleClient(h))
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(m),
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// Send back an error indicating that we couldn't join.
			p, err := stanza.NewPresence(*start)
			if err != nil {
				return err
			}
			p.Type = stanza.ErrorPresence
			p.To, p.From = p.From, p.To
			se := stanza.Error{
				By:        p.To.Bare(),
				Type:      stanza.Modify,
				Condition: stanza.NotAcceptable,
			}
			_, err = xmlstream.Copy(t, p.Wrap(xmlstream.MultiReader(
				xmlstream.Wrap(
					nil,
					xml.StartElement{Name: xml.Name{Space: muc.NS, Local: "x"}},
				),
				se.TokenReader(),
			)))
			return err
		}),
	)

	_, err := h.Join(context.Background(), j, s.Client)
	if !errors.Is(err, stanza.Error{}) {
		t.Fatalf("expected a stanza error but got: %v", err)
	}
}

func TestPartError(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	h := &muc.Client{}
	m := mux.New(ns.Client, muc.HandleClient(h))
	errHotelCalifornia := stanza.Error{
		Type:      stanza.Auth,
		Condition: stanza.NotAllowed,
	}
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(m),
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// Send back a self presence, indicating that the join is complete.
			p, err := stanza.NewPresence(*start)
			if err != nil {
				return err
			}
			p.To, p.From = p.From, p.To
			switch p.Type {
			case "":
				_, err = xmlstream.Copy(t, p.Wrap(xmlstream.Wrap(
					nil,
					xml.StartElement{Name: xml.Name{Space: muc.NSUser, Local: "x"}},
				)))
			case stanza.UnavailablePresence:
				p.Type = stanza.ErrorPresence
				_, err = xmlstream.Copy(t, p.Wrap(xmlstream.MultiReader(
					xmlstream.Wrap(
						nil,
						xml.StartElement{Name: xml.Name{Space: muc.NS, Local: "x"}},
					),
					errHotelCalifornia.TokenReader(),
				)))
			}
			return err
		}),
	)

	channel, err := h.Join(context.Background(), j, s.Client)
	if err != nil {
		t.Fatalf("error joining: %v", err)
	}

	err = channel.Leave(context.Background(), "")
	if !errors.Is(err, errHotelCalifornia) {
		t.Errorf("wrong error leaving: %v", err)
	}
	if channel.Joined() {
		t.Errorf("expected channel to be unjoined")
	}
}

func TestJoinCancel(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	s := xmpptest.NewClientServer()
	h := &muc.Client{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.Join(ctx, j, s.Client)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("wrong error: want=%v, got=%v", context.Canceled, err)
	}
}

func TestGetForm(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	const iqID = "1234"
	s := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			reply := `<iq type='result' id='` + iqID + `' to='me@localhost/cHKubP5q' from='` + j.Bare().String() + `'><query xmlns='http://jabber.org/protocol/muc#owner'><x type='form' xmlns='jabber:x:data'><title>Configuration</title><instructions>Complete and submit this form to configure the room.</instructions><field var='FORM_TYPE' type='hidden'><value>http://jabber.org/protocol/muc#roomconfig</value></field></x></query></iq>`
			d := xml.NewDecoder(strings.NewReader(reply))
			_, err := xmlstream.Copy(t, d)
			return err
		}),
	)

	formData, err := muc.GetConfigIQ(context.Background(), stanza.IQ{
		ID: iqID,
		To: j.Bare(),
	}, s.Client)
	if err != nil {
		t.Fatalf("error fetching form: %v", err)
	}

	const expected = "Configuration"
	if title := formData.Title(); title != expected {
		t.Errorf("wrong title, form decode failed: want=%q, got=%q", expected, title)
	}
}
