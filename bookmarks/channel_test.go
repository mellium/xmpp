// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks_test

import (
	"testing"

	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/internal/xmpptest"
)

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &bookmarks.Channel{},
		XML:   `<conference xmlns="urn:xmpp:bookmarks:1" autojoin="false"></conference>`,
	},
	1: {
		NoMarshal: true,
		Value:     &bookmarks.Channel{Autojoin: true},
		XML:       `<conference xmlns="urn:xmpp:bookmarks:1" autojoin="1"></conference>`,
	},
	2: {
		Value: &bookmarks.Channel{
			Autojoin:   true,
			Name:       "name",
			Nick:       "nick",
			Password:   "pass",
			Extensions: []byte("ext"),
		},
		XML: `<conference xmlns="urn:xmpp:bookmarks:1" autojoin="true" name="name"><nick>nick</nick><password>pass</password><extensions>ext</extensions></conference>`,
	},
	3: {
		Value: &bookmarks.Channel{
			Autojoin:   true,
			Name:       "name",
			Nick:       "nick",
			Password:   "pass",
			Extensions: []byte("ext"),
		},
		XML: `<conference xmlns="urn:xmpp:bookmarks:1" autojoin="true" name="name"><nick>nick</nick><password>pass</password><extensions>ext</extensions></conference>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
