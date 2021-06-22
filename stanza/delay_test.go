// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"testing"
	"time"

	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var marshalDelayTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value:       &stanza.Delay{},
		XML:         `<delay xmlns="urn:xmpp:delay" from="" stamp="0001-01-01T00:00:00Z"></delay>`,
		NoUnmarshal: true,
	},
	1: {
		Value: &stanza.Delay{
			From: jid.MustParse("example.net"),
		},
		XML: `<delay xmlns="urn:xmpp:delay" from="example.net" stamp="0001-01-01T00:00:00Z"></delay>`,
	},
	2: {
		Value: &stanza.Delay{
			From:  jid.MustParse("me@example.net"),
			Stamp: time.Unix(10000, 0).UTC(),
		},
		XML: `<delay xmlns="urn:xmpp:delay" from="me@example.net" stamp="1970-01-01T02:46:40Z"></delay>`,
	},
	3: {
		Value: &stanza.Delay{
			From:  jid.MustParse("me@example.net"),
			Stamp: time.Unix(10000, 0),
		},
		XML:         `<delay xmlns="urn:xmpp:delay" from="me@example.net" stamp="1970-01-01T02:46:40Z"></delay>`,
		NoUnmarshal: true,
	},
	4: {
		Value: &stanza.Delay{
			From:  jid.MustParse("me@example.net"),
			Stamp: time.Unix(10000, 0).UTC(),
		},
		XML:       `<delay xmlns="urn:xmpp:delay" from="me@example.net" stamp="1970-01-01T02:46:40Z"><foo/></delay>`,
		NoMarshal: true,
	},
	5: {
		Value: &stanza.Delay{
			From:   jid.MustParse("me@example.net"),
			Stamp:  time.Unix(10000, 0).UTC(),
			Reason: "test",
		},
		XML: `<delay xmlns="urn:xmpp:delay" from="me@example.net" stamp="1970-01-01T02:46:40Z">test</delay>`,
	},
}

func TestMarshalDelay(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalDelayTestCases)
}
