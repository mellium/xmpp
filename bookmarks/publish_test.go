// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)

func TestDelete(t *testing.T) {
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		err := e.EncodeToken(*start)
		if err != nil {
			return err
		}
		_, err = xmlstream.Copy(e, r)
		if err != nil {
			return err
		}
		return e.Flush()
	}))
	err := bookmarks.DeleteIQ(context.Background(), s.Client, stanza.IQ{
		ID: "123",
	}, s.Server.LocalAddr())
	if !errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable}) {
		t.Fatalf("error deleting bookmark: %v", err)
	}
	const expected = `<iq xmlns="jabber:client" xmlns="jabber:client" type="set" id="123"><pubsub xmlns="http://jabber.org/protocol/pubsub" xmlns="http://jabber.org/protocol/pubsub"><retract xmlns="http://jabber.org/protocol/pubsub" node="urn:xmpp:bookmarks:1" notify="true"><item xmlns="http://jabber.org/protocol/pubsub" id="example.net"></item></retract></pubsub></iq>`
	if s := buf.String(); s != expected {
		t.Fatalf("Wrong XML:\nwant=%s\n got=%s", expected, s)
	}
}
