// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"testing"
)

// Compile time check ot make sure that JID and *JID match several interfaces.
var _ fmt.Stringer = (*JID)(nil)
var _ xml.MarshalerAttr = (*JID)(nil)
var _ xml.UnmarshalerAttr = (*JID)(nil)
var _ xml.Marshaler = (*JID)(nil)
var _ xml.Unmarshaler = (*JID)(nil)
var _ net.Addr = (*JID)(nil)

func TestValidJIDs(t *testing.T) {
	for _, jid := range []struct {
		jid, lp, dp, rp string
	}{
		{"example.net", "", "example.net", ""},
		{"example.net/rp", "", "example.net", "rp"},
		{"mercutio@example.net", "mercutio", "example.net", ""},
		{"mercutio@example.net/rp", "mercutio", "example.net", "rp"},
		{"mercutio@example.net/rp@rp", "mercutio", "example.net", "rp@rp"},
		{"mercutio@example.net/rp@rp/rp", "mercutio", "example.net", "rp@rp/rp"},
		{"mercutio@example.net/@", "mercutio", "example.net", "@"},
		{"mercutio@example.net//@", "mercutio", "example.net", "/@"},
		{"mercutio@example.net//@//", "mercutio", "example.net", "/@//"},
		{"[::1]", "", "[::1]", ""},
	} {
		j, err := Parse(jid.jid)
		switch {
		case err != nil:
			t.Error(err)
		case j.Domainpart() != jid.dp:
			t.Errorf("Got domainpart %s but expected %s", j.Domainpart(), jid.dp)
		case j.Localpart() != jid.lp:
			t.Errorf("Got localpart %s but expected %s", j.Localpart(), jid.lp)
		case j.Resourcepart() != jid.rp:
			t.Errorf("Got resourcepart %s but expected %s", j.Resourcepart(), jid.rp)
		}
	}
}

var invalidutf8 = string([]byte{0xff, 0xfe, 0xfd})

func TestInvalidParseJIDs(t *testing.T) {

	for _, jid := range []string{
		"test@/test",
		invalidutf8 + "@example.com/rp",
		invalidutf8 + "/rp",
		invalidutf8,
		"example.com/" + invalidutf8,
		"lp@/rp",
		`b"d@example.net`,
		`b&d@example.net`,
		`b'd@example.net`,
		`b:d@example.net`,
		`b<d@example.net`,
		`b>d@example.net`,
		`e@example.net/`,
	} {
		_, err := Parse(jid)
		if err == nil {
			t.Errorf("Expected JID %s to fail", jid)
		}
	}
}

func TestInvalidNewJIDs(t *testing.T) {
	for _, jid := range []struct {
		lp, dp, rp string
	}{
		{strings.Repeat("a", 1024), "example.net", ""},
		{"e", "example.net", strings.Repeat("a", 1024)},
		{"b/d", "example.net", ""},
		{"b@d", "example.net", ""},
		{"e", "[example.net]", ""},
	} {
		_, err := New(jid.lp, jid.dp, jid.rp)
		if err == nil {
			t.Errorf("Expected composition of JID parts %s to fail", jid)
		}
	}
}

func TestMarshalAttrEmpty(t *testing.T) {
	attr, err := ((*JID)(nil)).MarshalXMLAttr(xml.Name{})
	switch {
	case err != nil:
		t.Logf("Marshaling an empty JID to an attr should not error but got %v\n", err)
		t.Fail()
	case attr != xml.Attr{}:
		t.Logf("Error marshaling empty JID expected Attr{} but got: %+v\n", err)
		t.Fail()
	}
}

func TestMustParsePanics(t *testing.T) {
	handleErr := func(shouldPanic bool) {
		r := recover()
		switch {
		case shouldPanic && r == nil:
			t.Error("Must parse should panic on invalid JID")
		case !shouldPanic && r != nil:
			t.Error("Must parse should not panic on valid JID")
		}
	}
	for _, t := range []struct {
		jid         string
		shouldPanic bool
	}{
		{"@me", true},
		{"@`me", true},
		{"e@example.net", false},
	} {
		func() {
			defer handleErr(t.shouldPanic)
			MustParse(t.jid)
		}()
	}
}

func TestEqual(t *testing.T) {
	m := MustParse("mercutio@example.net/test")
	for _, test := range []struct {
		j1, j2 *JID
		eq     bool
	}{
		{m, MustParse("mercutio@example.net/test"), true},
		{m.Bare(), MustParse("mercutio@example.net"), true},
		{m.Domain(), MustParse("example.net"), true},
		{m, MustParse("mercutio@example.net/nope"), false},
		{m, MustParse("mercutio@e.com/test"), false},
		{m, MustParse("m@example.net/test"), false},
		{(*JID)(nil), (*JID)(nil), true},
		{m, (*JID)(nil), false},
		{(*JID)(nil), m, false},
	} {
		switch {
		case test.eq && !test.j1.Equal(test.j2):
			t.Errorf("JIDs %s and %s should be equal", test.j1, test.j2)
		case !test.eq && test.j1.Equal(test.j2):
			t.Errorf("JIDs %s and %s should not be equal", test.j1, test.j2)
		}
	}
}

func TestNetwork(t *testing.T) {
	if MustParse("test").Network() != "xmpp" {
		t.Error("Network should be `xmpp`")
	}
}

func TestCopy(t *testing.T) {
	m := MustParse("mercutio@example.net/test")
	m2 := m.Copy()
	switch {
	case !m.Equal(m2):
		t.Error("Copying a JID should still result in equal JIDs")
	case m == m2:
		t.Error("Copying a JID should result in a different JID pointer")
	}
}

const allescaped = `\20\22\26\27\2f\3a\3c\3e\40\5c`

func TestEscape(t *testing.T) {
	for _, test := range []struct {
		unescaped, escaped string
	}{
		{escape, allescaped},
		{`nothingtodohere`, `nothingtodohere`},
		{"", ""},
	} {
		if e := Escape(test.unescaped); e != test.escaped {
			t.Errorf("Escaped localpart should be `%s` but got: `%s`", test.escaped, e)
		}
	}
}

func TestUnescape(t *testing.T) {
	for _, test := range []struct {
		escaped, unescaped string
	}{
		{allescaped, escape},
		{`\20\3c\3C\aa\\\`, ` <<\aa\\\`},
		{"nothingtodohere", "nothingtodohere"},
		{"", ""},
	} {
		if u := Unescape(test.escaped); u != test.unescaped {
			t.Errorf("Unescaped localpart should be `%s` but got: `%s`", test.unescaped, u)
		}
	}
}

func TestMarshalXML(t *testing.T) {
	// Test default marshaling
	j := MustParse("feste@shakespeare.lit")
	b, err := xml.Marshal(j)
	switch expected := `<JID>feste@shakespeare.lit</JID>`; {
	case err != nil:
		t.Error(err)
	case string(b) != expected:
		t.Errorf("Error marshaling JID, expected `%s` but got `%s`", expected, string(b))
	}

	// Test encoding with a custom element
	j = MustParse("feste@shakespeare.lit/ilyria")
	var buf bytes.Buffer
	start := xml.StartElement{Name: xml.Name{Space: "", Local: "item"}, Attr: []xml.Attr{}}
	e := xml.NewEncoder(&buf)
	err = e.EncodeElement(j, start)
	switch expected := `<item>feste@shakespeare.lit/ilyria</item>`; {
	case err != nil:
		t.Error(err)
	case buf.String() != expected:
		t.Errorf("Error encoding JID, expected `%s` but got `%s`", expected, buf.String())
	}

	// Test encoding a nil JID
	j = (*JID)(nil)
	b, err = xml.Marshal(j)
	switch expected := ``; {
	case err != nil:
		t.Error(err)
	case string(b) != expected:
		t.Errorf("Error marshaling JID, expected `%s` but got `%s`", expected, string(b))
	}
}

func TestUnmarshal(t *testing.T) {
	for _, test := range []struct {
		xml string
		jid *JID
		err bool
	}{
		{`<item>feste@shakespeare.lit/ilyria</item>`, MustParse("feste@shakespeare.lit/ilyria"), false},
		{`<jid>feste@shakespeare.lit</jid>`, MustParse("feste@shakespeare.lit"), false},
		{`<oops>feste@shakespeare.lit</bad>`, nil, true},
		{`<item></item>`, nil, true},
	} {
		j := &JID{}
		err := xml.Unmarshal([]byte(test.xml), j)
		switch {
		case test.err && err == nil:
			t.Errorf("Expected unmarshaling `%s` as a JID to return an error", test.xml)
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case !test.jid.Equal(j):
			t.Errorf("Expected JID to unmarshal to `%s` but got `%s`", test.jid, j)
		}
	}
}
