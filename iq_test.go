// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"testing"
)

func TestMarshalIQTypeAttr(t *testing.T) {
	n := xml.Name{Space: "space", Local: "type"}
	for _, test := range []struct {
		iqtype iqType
		value  string
	}{{Get, "get"}, {Set, "set"}, {Result, "result"}, {Error, "error"}} {
		attr, err := test.iqtype.MarshalXMLAttr(n)
		if err != nil {
			t.Error(err)
			continue
		}
		if attr.Name != n {
			t.Errorf("Got wrong attribute name for IQ type %s. Got %v, want %v", test.value, attr.Name, n)
		}
		if attr.Value != test.value {
			t.Errorf("Got wrong attribute value for IQ type %s: `%s`", test.value, attr.Value)
		}
	}
}
