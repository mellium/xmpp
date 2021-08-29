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
	_ xml.Marshaler       = info.Identity{}
	_ xmlstream.Marshaler = info.Identity{}
	_ xmlstream.WriterTo  = info.Identity{}
)

func TestEncodeIdentity(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		0: {
			Value:       &info.Identity{},
			XML:         `<identity xmlns="http://jabber.org/protocol/disco#info" category="" type=""></identity>`,
			NoUnmarshal: true,
		},
		1: {
			Value: &info.Identity{
				XMLName:  xml.Name{Space: disco.NSInfo, Local: "identity"},
				Category: "cat",
				Type:     "typ",
				Name:     "name",
			},
			XML: `<identity xmlns="http://jabber.org/protocol/disco#info" category="cat" type="typ" name="name"></identity>`,
		},
	})
}
