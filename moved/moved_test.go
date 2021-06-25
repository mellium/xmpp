// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package moved_test

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/moved"
	"mellium.im/xmpp/stanza"
)

func TestGetForm(t *testing.T) {
	j := jid.MustParse("room@example.net/me")
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(m),
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			reply := `<iq type='result' id='` + iqID + `' to='me@localhost/cHKubP5q' from='` + j.Bare().String() + `'><query xmlns='http://jabber.org/protocol/muc#owner'><x type='form' xmlns='jabber:x:data'><title>Configuration</title><instructions>Complete and submit this form to configure the room.</instructions><field var='FORM_TYPE' type='hidden'><value>http://jabber.org/protocol/muc#roomconfig</value></field></x></query></iq>`
			d := xml.NewDecoder(strings.NewReader(reply))
			_, err := xmlstream.Copy(t, d)
			return err
		}),
	)

	formData, err := h.GetConfigIQ(context.Background(), stanza.IQ{
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
