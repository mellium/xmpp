// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.
package jid

import (
	"bytes"
	"encoding/xml"
	"testing"
)

// Ensure that JID parts are split properly.
func TestValidPartsFromString(t *testing.T) {
	decompositions := [][]string{
		{"lp@dp/rp", "lp", "dp", "rp"},
		{"dp/rp", "", "dp", "rp"},
		{"dp", "", "dp", ""},
		{"lp@dp//rp", "lp", "dp", "/rp"},
		{"lp@dp/rp/", "lp", "dp", "rp/"},
		{"lp@dp/@rp/", "lp", "dp", "@rp/"},
		{"lp@dp/lp@dp/rp", "lp", "dp", "lp@dp/rp"},
		{"dp//rp", "", "dp", "/rp"},
		{"dp/rp/", "", "dp", "rp/"},
		{"dp/@rp/", "", "dp", "@rp/"},
		{"dp/lp@dp/rp", "", "dp", "lp@dp/rp"},
		{"₩", "", "₩", ""},
	}
	for _, d := range decompositions {
		lp, dp, rp, err := partsFromString(d[0])
		if err != nil || lp != d[1] || dp != d[2] || rp != d[3] {
			t.FailNow()
		}
	}
}

func TestValidFromParts(t *testing.T) {
	decompositions := [][]string{
		{"lp", "dp", "rp", "lp", "dp", "rp"},
		{"ｌｐ", "ｄｐ", "ｒｐ", "lp", "dp", "ｒｐ"},
		{"ﾛ", "ﾛ", "ﾛ", "ロ", "ロ", "ﾛ"},
		{"", "127.0.0.1", "", "", "127.0.0.1", ""},
		{"", "[::1]", "", "", "[::1]", ""},
	}
	for _, d := range decompositions {
		j, err := FromParts(d[0], d[1], d[2])
		if err != nil || j.localpart != d[3] || j.domainpart != d[4] ||
			j.resourcepart != d[5] {
			t.FailNow()
		}
	}
}

func TestInvalidFromParts(t *testing.T) {
	decompositions := [][]string{
		{"lp", "", "rp"},
		{"", "[test]", ""},
		{"", "[127.0.0.1]", ""},
		{"", "\u0660", ""},
		{"", "\u0669", ""},
		{"", "\u303B", ""},
		// Currently failing:
		{"lp", "@test", "rp"},
		{"lp", "test/", "rp"},
		{"", "::1", ""},
		{"lp ", "example.com", "rp"},
		{"lp\t", "example.com", "rp"},
		{"lp", " example.com", "rp"},
		{"lp", "\texample.com", "rp"},
	}
	for _, d := range decompositions {
		if _, err := FromParts(d[0], d[1], d[2]); err == nil {
			t.FailNow()
		}
	}
}

// Ensure that JIDs that are too long return an error.
func TestLongParts(t *testing.T) {
	// Generate a part that is too long.
	pb := bytes.NewBuffer(make([]byte, 0, 1024))
	for i := 0; i < 64; i++ {
		pb.WriteString("aaaaaaaaaaaaaaaa")
	}
	ps := pb.String()
	jids := []string{
		ps + "@example.com/test",
		"lp@" + ps + "/test",
		"lp@example.com/" + ps,
		ps + "@" + ps + "/" + ps,
	}
	for _, d := range jids {
		if _, _, _, err := partsFromString(d); err != nil {
			t.FailNow()
		}
	}
}

// JIDS cannot contain invalid UTF8 in the localpart.
func TestNewInvalidUtf8Localpart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := FromString(invalid + "@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// JIDS cannot contain invalid UTF8 in the domainpart.
func TestNewInvalidUtf8Domainpart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := FromString("example@" + invalid + "/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// JIDS cannot contain invalid UTF8 in the resourcepart.
func TestNewInvalidUtf8Resourcepart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := FromString("example@example.com/" + invalid)
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with an empty localpart should error.
func TestNewEmptyLocalpart(t *testing.T) {
	_, err := FromString("@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with no localpart should work.
func TestNewNoLocalpart(t *testing.T) {
	jid, err := FromString("example.com/resourcepart")
	if err != nil || jid.Localpart() != "" {
		t.FailNow()
	}
}

// Trying to create a JID with no domainpart should error.
func TestNewNoDomainpart(t *testing.T) {
	_, err := FromString("text@/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with no anything should error.
func TestNewNoAnything(t *testing.T) {
	_, err := FromString("@/")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID from an empty string should error.
func TestNewEmptyString(t *testing.T) {
	_, err := FromString("")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with '@' or '/' in the resourcepart should work.
func TestNewJidInResourcepart(t *testing.T) {
	_, err := FromString("this/is@/fine")
	if err != nil {
		t.FailNow()
	}
}

// Trying to create a JID with an empty resource part should error.
func TestNewEmptyResourcepart(t *testing.T) {
	_, err := FromString("text@example.com/")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a new bare JID (no resource part) should work.
func TestNewBareJid(t *testing.T) {
	jid, err := FromString("barejid@example.com")
	if err != nil || jid.Resourcepart() != "" {
		t.FailNow()
	}
}

// Creating a new JID from a valid JID string should work and contain all the
// correct parts.
func TestNewValid(t *testing.T) {
	s := "jid@example.com/resourcepart"
	jid, err := FromString(s)
	if err != nil {
		t.FailNow()
	}
	switch {
	case err != nil:
		t.FailNow()
	case jid.Localpart() != "jid":
		t.FailNow()
	case jid.Domainpart() != "example.com":
		t.FailNow()
	case jid.Resourcepart() != "resourcepart":
		t.FailNow()
	}
}

// Two identical JIDs should be equal.
func TestEqualJIDs(t *testing.T) {
	jid := &Jid{"newjid", "example.com", "equal", false}
	jid2 := &Jid{"newjid", "example.com", "equal", false}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// A Jid constructed from another Jid should be equal to the original.
func TestFromJid(t *testing.T) {
	// Check that Jids that are validated but don't change match
	j := &Jid{"newjid", "example.com", "equal", false}
	jv, err := FromJid(j)
	if err != nil || !j.Equals(jv) {
		t.FailNow()
	}

	// Check that Jids which are validated and changed don't match
	j = &Jid{"\u212akelvinsign", "example.com", "equal", false}
	jv, err = FromJid(j)
	if err != nil || j.Equals(jv) {
		t.FailNow()
	}

	// Check that already valid Jid's still match
	j = &Jid{"newjid", "example.com", "equal", true}
	jv, err = FromJid(j)
	if err != nil || !j.Equals(jv) {
		t.FailNow()
	}
}

// A Jid should equal a copy of itself.
func TestCopy(t *testing.T) {
	j := &Jid{"newjid", "example.com", "equal", false}
	if !j.Equals(j.Copy()) {
		t.FailNow()
	}
}

// Two different JIDs should not be equal.
func TestNotEqualJIDs(t *testing.T) {
	jid := &Jid{"newjid", "example.com", "notequal", false}
	jid2 := &Jid{"newjid2", "example.com", "notequal", false}
	if jid.Equals(jid2) {
		t.FailNow()
	}
	jid = &Jid{"newjid", "example.com", "notequal", false}
	jid2 = &Jid{"newjid", "example.net", "notequal", false}
	if jid.Equals(jid2) {
		t.FailNow()
	}
	jid = &Jid{"newjid", "example.com", "notequal", false}
	jid2 = &Jid{"newjid", "example.com", "notequal2", false}
	if jid.Equals(jid2) {
		t.FailNow()
	}
}

// The localpart should be normalized.
func TestEqualsUnicodeNormLocalpart(t *testing.T) {
	// U+2126 Ω ohm sign
	jid, err := FromString("\u2126@example.com/res")
	if err != nil {
		t.Fail()
	}
	// U+03A9 Ω greek capital letter omega
	jid2, err := FromString("\u03a9@example.com/res")
	if err != nil {
		t.Fail()
	}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// The resourcepart should be normalized.
func TestEqualsUnicodeNormResourcepart(t *testing.T) {
	// U+2126 Ω ohm sign
	jid, err := FromString("example@example.com/res\u2126")
	if err != nil {
		t.Fail()
	}
	// U+03A9 Ω greek capital letter omega
	jid2, err := FromString("example@example.com/res\u03a9")
	if err != nil {
		t.Fail()
	}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// Jids should be marshalable to an XML attribute
func TestMarshal(t *testing.T) {
	jid := Jid{"newjid", "example.com", "marshal", false}
	attr, err := jid.MarshalXMLAttr(xml.Name{Space: "", Local: "to"})

	if err != nil || attr.Name.Local != "to" || attr.Name.Space != "" || attr.Value != "newjid@example.com/marshal" {
		t.FailNow()
	}
}
