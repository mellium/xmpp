// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package attr_test

import (
	"encoding/xml"
	"strconv"
	"testing"

	"mellium.im/xmpp/internal/attr"
)

var attrTests = [...]struct {
	attr  []xml.Attr
	local string
	out   string
}{
	0: {},
	1: {local: "test"},
	2: {attr: []xml.Attr{}},
	3: {attr: []xml.Attr{}, local: "test"},
	4: {
		attr:  []xml.Attr{{Name: xml.Name{Local: "test"}, Value: "test"}},
		local: "test",
		out:   "test",
	},
	5: {
		attr: []xml.Attr{
			{Name: xml.Name{Local: "test"}, Value: "test0"},
			{Name: xml.Name{Local: "test"}, Value: "test1"},
		},
		local: "test",
		out:   "test0",
	},
}

func TestAttr(t *testing.T) {
	for i, tc := range attrTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := attr.Get(tc.attr, tc.local)
			if out != tc.out {
				t.Errorf("Wrong output: want=%q, got=%q", tc.out, out)
			}
		})
	}
}
