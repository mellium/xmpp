// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmpp/internal/xmpptest"
)

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		NoMarshal: true,
		Value:     &struct{ Foo int }{Foo: 0},
		XML:       `<Foo>0</Foo>`,
	},
	1: {
		NoUnmarshal: true,
		Value: &struct {
			XMLName xml.Name `xml:"foo"`
			Foo     int      `xml:",chardata"`
		}{Foo: 0},
		XML: `<foo>0</foo>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
