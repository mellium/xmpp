// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"fmt"
	"testing"
)

var (
	_ xml.UnmarshalerAttr = (*iqType)(nil)
	_ xml.MarshalerAttr   = (*iqType)(nil)
	_ xml.MarshalerAttr   = GetIQ
	_ fmt.Stringer        = (*iqType)(nil)
	_ fmt.Stringer        = GetIQ
)

func TestMarshalIQTypeAttr(t *testing.T) {
	n := xml.Name{Space: "space", Local: "type"}
	for _, test := range []struct {
		iqtype iqType
		value  string
	}{{GetIQ, "get"}, {SetIQ, "set"}, {ResultIQ, "result"}, {ErrorIQ, "error"}} {
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

func TestUnmarshalIQTypeAttr(t *testing.T) {
	for _, test := range []struct {
		attr   xml.Attr
		iqtype iqType
		err    bool
	}{
		{xml.Attr{Name: xml.Name{}, Value: "get"}, GetIQ, false},
		{xml.Attr{Name: xml.Name{Space: "", Local: "type"}, Value: "set"}, SetIQ, false},
		{xml.Attr{Name: xml.Name{Space: "urn", Local: "loc"}, Value: "result"}, ResultIQ, false},
		{xml.Attr{Name: xml.Name{}, Value: "error"}, ErrorIQ, false},
		{xml.Attr{Name: xml.Name{}, Value: "stuff"}, ErrorIQ, true},
	} {
		iqtype := iqType(0)
		switch err := (&iqtype).UnmarshalXMLAttr(test.attr); {
		case test.err && err == nil:
			t.Error("Expected unmarshaling IQ type to error")
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case test.err && err != nil:
			continue
		case iqtype != test.iqtype:
			t.Errorf("Expected attr %+v to unmarshal into %s type IQ but got %s", test.attr, test.iqtype, iqtype)
		}
	}
}
