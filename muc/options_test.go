// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc

import (
	"testing"
	"time"

	"mellium.im/xmpp/internal/xmpptest"
)

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: func() *config {
			c := &config{}
			MaxHistory(1)(c)
			MaxBytes(2)(c)
			Duration(3 * time.Second)(c)
			Since(time.Time{})(c)
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxstanzas="1" maxchars="2" seconds="3" since="0001-01-01T00:00:00Z"></history><password>test</password></x>`,
	},
	1: {
		Value: func() *config {
			c := &config{}
			MaxHistory(1)(c)
			MaxBytes(2)(c)
			Duration(3 * time.Second)(c)
			Since(time.Time{})(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxstanzas="1" maxchars="2" seconds="3" since="0001-01-01T00:00:00Z"></history></x>`,
	},
	2: {
		Value: func() *config {
			c := &config{}
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><password>test</password></x>`,
	},
	3: {
		Value: &config{},
		XML:   `<x xmlns="http://jabber.org/protocol/muc"></x>`,
	},
	4: {
		Value: func() *config {
			c := &config{}
			MaxBytes(2)(c)
			Duration(3 * time.Second)(c)
			Since(time.Time{})(c)
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxchars="2" seconds="3" since="0001-01-01T00:00:00Z"></history><password>test</password></x>`,
	},
	5: {
		Value: func() *config {
			c := &config{}
			MaxHistory(1)(c)
			Duration(3 * time.Second)(c)
			Since(time.Time{})(c)
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxstanzas="1" seconds="3" since="0001-01-01T00:00:00Z"></history><password>test</password></x>`,
	},
	6: {
		Value: func() *config {
			c := &config{}
			MaxHistory(1)(c)
			MaxBytes(2)(c)
			Since(time.Time{})(c)
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxstanzas="1" maxchars="2" since="0001-01-01T00:00:00Z"></history><password>test</password></x>`,
	},
	7: {
		Value: func() *config {
			c := &config{}
			MaxHistory(1)(c)
			MaxBytes(2)(c)
			Duration(3 * time.Second)(c)
			Password("test")(c)
			return c
		}(),
		XML: `<x xmlns="http://jabber.org/protocol/muc"><history maxstanzas="1" maxchars="2" seconds="3"></history><password>test</password></x>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
