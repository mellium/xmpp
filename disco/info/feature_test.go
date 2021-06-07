// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package info_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/internal/xmpptest"
)

var (
	_ xml.Marshaler       = info.Feature{}
	_ xmlstream.Marshaler = info.Feature{}
	_ xmlstream.WriterTo  = info.Feature{}
)

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		0: {
			Value:       &info.Feature{},
			XML:         `<feature xmlns="http://jabber.org/protocol/disco#info" var=""></feature>`,
			NoUnmarshal: true,
		},
		1: {
			Value: &info.Feature{
				XMLName: xml.Name{Space: disco.NSInfo, Local: "feature"},
				Var:     "urn:example",
			},
			XML: `<feature xmlns="http://jabber.org/protocol/disco#info" var="urn:example"></feature>`,
		},
	})
}
