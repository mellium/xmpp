// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// Compile time interface checks.
var _ fmt.Stringer = &Version{}
var _ fmt.Stringer = Version{}
var _ xml.MarshalerAttr = &Version{}
var _ xml.MarshalerAttr = Version{}
var _ xml.UnmarshalerAttr = (*Version)(nil)

// Strings must parse correctly.
func TestParseVersion(t *testing.T) {
	for _, data := range []struct {
		vs        string
		v         Version
		shouldErr bool
	}{
		{"1.0", Version{1, 0}, false},
		{"1.0.0", Version{}, true},
		{"A.1", Version{}, true},
		{"1.a", Version{}, true},
		{"1.0xA", Version{}, true},
		{"", Version{}, true},
	} {
		v, err := ParseVersion(data.vs)
		switch {
		case data.shouldErr && err == nil:
			t.Logf("Version '%s' should fail with an error when parsed.", data.vs)
			t.Fail()
		case !data.shouldErr && err != nil:
			t.Logf("Error encountered while parsing '%s': %v", data.vs, err)
			t.Fail()
		case data.shouldErr && err != nil:
			continue
		case !data.shouldErr && err == nil:
			if v != data.v {
				t.Logf("Parsing version %s expected %v but got %v", data.vs, data.v, v)
				t.Fail()
			}
			continue
		}
	}
}

func TestMustParseVersionPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustParseVersion to panic when given invalid version")
		}
	}()
	MustParseVersion("a.0")
}

func TestCompareVersion(t *testing.T) {
	for _, data := range []struct {
		v1, v2 Version
		less   bool
	}{
		{Version{}, Version{}, false},
		{MustParseVersion("1.0"), MustParseVersion("1.1"), true},
		{MustParseVersion("1.1"), MustParseVersion("1.0"), false},
		{MustParseVersion("1.0"), MustParseVersion("2.0"), true},
		{MustParseVersion("2.0"), MustParseVersion("1.0"), false},
		{MustParseVersion("1.5"), MustParseVersion("2.0"), true},
		{MustParseVersion("2.0"), MustParseVersion("1.5"), false},
	} {
		if data.v1.Less(data.v2) != data.less {
			if data.less {
				t.Errorf("Expected %v to be less than %v", data.v1, data.v2)
			} else {
				t.Errorf("Expected %v to be greater than %v", data.v1, data.v2)
			}
		}
	}
}

func TestMarshalVersion(t *testing.T) {
	n := xml.Name{Space: "", Local: "test"}
	for _, data := range []string{
		"1.0", "0.1", "10.0", "0.10",
	} {
		switch s2, _ := MustParseVersion(data).MarshalXMLAttr(n); {
		case s2.Value != data:
			t.Errorf("Expected %s to parse and stringify to itself but got %s", data, s2)
		case s2.Name != n:
			t.Errorf("Expected %s to marshal to an attribute with name %v but got %v", data, n, s2.Name)
		}
	}
}

func TestUnmarshalVersion(t *testing.T) {
	for _, data := range []struct {
		attr xml.Attr
		v    string
		err  bool
	}{
		{xml.Attr{}, "", true},
		{xml.Attr{Value: "2.0"}, "2.0", false},
		{xml.Attr{Name: xml.Name{Space: "", Local: "Whatever"}, Value: "0.9"}, "0.9", false},
	} {
		v2 := Version{}
		err := v2.UnmarshalXMLAttr(data.attr)
		switch {
		case data.err && err == nil:
			t.Errorf("Expected unmarshaling %v to error", data.attr)
			continue
		case !data.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case v2.String() != data.v:
			t.Errorf("Expected unmarshaled attr %v to equal %s but got %s", data.attr, data.v, v2.String())
		}

	}
}
