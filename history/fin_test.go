// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/paging"
)

var (
	_ xml.Unmarshaler     = (*history.Query)(nil)
	_ xml.Marshaler       = (*history.Query)(nil)
	_ xmlstream.Marshaler = (*history.Query)(nil)
	_ xmlstream.WriterTo  = (*history.Query)(nil)
)

var resEncodingTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &history.Result{
			Set: paging.Set{
				XMLName: xml.Name{Space: paging.NS, Local: "set"},
			},
		},
		XML: `<fin xmlns="urn:xmpp:mam:2" complete="false" stable="true"><set xmlns="http://jabber.org/protocol/rsm"><first></first><last></last></set></fin>`,
	},
}

func TestEncodeResult(t *testing.T) {
	xmpptest.RunEncodingTests(t, resEncodingTestCases)
}
