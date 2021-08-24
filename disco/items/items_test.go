// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package items_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
)

var (
	_ xml.Marshaler       = items.Item{}
	_ xmlstream.Marshaler = items.Item{}
	_ xmlstream.WriterTo  = items.Item{}
)

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		0: {
			Value:       &items.Item{},
			XML:         `<item xmlns="http://jabber.org/protocol/disco#items" jid=""></item>`,
			NoUnmarshal: true,
		},
		1: {
			Value: &items.Item{
				XMLName: xml.Name{Space: disco.NSItems, Local: "item"},
				JID:     jid.MustParse("example.net"),
				Node:    "urn:example",
				Name:    "test",
			},
			XML: `<item xmlns="http://jabber.org/protocol/disco#items" jid="example.net" node="urn:example" name="test"></item>`,
		},
	})
}
