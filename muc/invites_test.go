// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
)

var (
	_ xml.Marshaler       = muc.Invitation{}
	_ xml.Unmarshaler     = (*muc.Invitation)(nil)
	_ xmlstream.Marshaler = muc.Invitation{}
	_ xmlstream.WriterTo  = muc.Invitation{}
)

var inviteEncodingTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value:       &muc.Invitation{},
		XML:         `<x xmlns="http://jabber.org/protocol/muc#user"><invite to=""></invite></x>`,
		NoUnmarshal: true,
	},
	1: {
		Value: &muc.Invitation{XMLName: xml.Name{Space: muc.NSUser, Local: "x"}},
		XML:   `<x xmlns="http://jabber.org/protocol/muc#user"><invite to=""></invite></x>`,
	},
	2: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSUser, Local: "x"},
			Continue: true,
			Thread:   "123",
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
			Password: "NCC-1701-D",
			Reason:   "Senior officers to the bridge.",
		},
		XML: `<x xmlns="http://jabber.org/protocol/muc#user"><invite to="bridgecrew@muc.localhost"><reason>Senior officers to the bridge.</reason><continue thread="123"></continue></invite><password>NCC-1701-D</password></x>`,
	},
	3: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSUser, Local: "x"},
			Thread:   "123",
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
			Password: "NCC-1701-D",
			Reason:   "Senior officers to the bridge.",
		},
		XML:         `<x xmlns="http://jabber.org/protocol/muc#user"><invite to="bridgecrew@muc.localhost"><reason>Senior officers to the bridge.</reason></invite><password>NCC-1701-D</password></x>`,
		NoUnmarshal: true,
	},
	4: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSUser, Local: "x"},
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
			Continue: true,
		},
		XML: `<x xmlns="http://jabber.org/protocol/muc#user"><invite to="bridgecrew@muc.localhost"><continue></continue></invite></x>`,
	},

	5: {
		Value: &muc.Invitation{XMLName: xml.Name{Space: muc.NSConf, Local: "x"}},
		XML:   `<x xmlns="jabber:x:conference" jid=""></x>`,
	},
	6: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSConf, Local: "x"},
			Continue: true,
			Thread:   "123",
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
			Password: "NCC-1701-D",
			Reason:   "Senior officers to the bridge.",
		},
		XML: `<x xmlns="jabber:x:conference" jid="bridgecrew@muc.localhost" continue="true" thread="123" password="NCC-1701-D" reason="Senior officers to the bridge."></x>`,
	},
	7: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSConf, Local: "x"},
			Thread:   "123",
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
			Password: "NCC-1701-D",
			Reason:   "Senior officers to the bridge.",
		},
		XML:         `<x xmlns="jabber:x:conference" jid="bridgecrew@muc.localhost" password="NCC-1701-D" reason="Senior officers to the bridge."></x>`,
		NoUnmarshal: true,
	},
	8: {
		Value: &muc.Invitation{
			XMLName:  xml.Name{Space: muc.NSConf, Local: "x"},
			Continue: true,
			JID:      jid.MustParse("bridgecrew@muc.localhost"),
		},
		XML: `<x xmlns="jabber:x:conference" jid="bridgecrew@muc.localhost" continue="true"></x>`,
	},
}

func TestActions(t *testing.T) {
	xmpptest.RunEncodingTests(t, inviteEncodingTestCases)
}
