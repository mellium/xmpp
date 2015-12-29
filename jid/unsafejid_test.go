// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"bytes"
	"encoding/xml"
	"testing"
)

// Compile time check ot make sure that UnsafeJID is a JID
var _ JID = (*UnsafeJID)(nil)

// Ensure that JID parts are split properly.
func TestValidPartsFromString(t *testing.T) {
	for _, d := range [][]string{
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
	} {
		lp, dp, rp, err := SplitString(d[0])
		if err != nil || lp != d[1] || dp != d[2] || rp != d[3] {
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
		if _, _, _, err := SplitString(d); err != nil {
			t.FailNow()
		}
	}
}

// Trying to create a JID with an empty localpart should error.
func TestNewEmptyLocalpart(t *testing.T) {
	_, err := UnsafeFromString("@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with no localpart should work.
func TestNewNoLocalpart(t *testing.T) {
	jid, err := UnsafeFromString("example.com/resourcepart")
	if err != nil || jid.Localpart() != "" {
		t.FailNow()
	}
}

// Trying to create a JID with no domainpart should error.
func TestNewNoDomainpart(t *testing.T) {
	_, err := UnsafeFromString("text@/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with no anything should error.
func TestNewNoAnything(t *testing.T) {
	_, err := UnsafeFromString("@/")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID from an empty string should error.
func TestNewEmptyString(t *testing.T) {
	_, err := UnsafeFromString("")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with '@' or '/' in the resourcepart should work.
func TestNewJIDInResourcepart(t *testing.T) {
	_, err := UnsafeFromString("this/is@/fine")
	if err != nil {
		t.FailNow()
	}
}

// Trying to create a JID with an empty resource part should error.
func TestNewEmptyResourcepart(t *testing.T) {
	_, err := UnsafeFromString("text@example.com/")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a new bare JID (no resource part) should work.
func TestNewBareUnsafeJID(t *testing.T) {
	jid, err := UnsafeFromString("barejid@example.com")
	if err != nil || jid.Resourcepart() != "" {
		t.FailNow()
	}
}

// Creating a new JID from a valid JID string should work and contain all the
// correct parts.
func TestNewValid(t *testing.T) {
	s := "jid@example.com/resourcepart"
	jid, err := UnsafeFromString(s)
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
	jid := &UnsafeJID{"newjid", "example.com", "equal"}
	jid2 := &UnsafeJID{"newjid", "example.com", "equal"}
	if !jid.Equal(jid2) {
		t.FailNow()
	}
}

// An UnsafeJID should equal a copy of itself.
func TestCopy(t *testing.T) {
	j := &UnsafeJID{"newjid", "example.com", "equal"}
	if !j.Equal(j.Copy()) {
		t.FailNow()
	}
}

// Two different JIDs should not be equal.
func TestNotEqualJIDs(t *testing.T) {
	jid := &UnsafeJID{"newjid", "example.com", "notequal"}
	jid2 := &UnsafeJID{"newjid2", "example.com", "notequal"}
	if jid.Equal(jid2) {
		t.FailNow()
	}
	jid = &UnsafeJID{"newjid", "example.com", "notequal"}
	jid2 = &UnsafeJID{"newjid", "example.net", "notequal"}
	if jid.Equal(jid2) {
		t.FailNow()
	}
	jid = &UnsafeJID{"newjid", "example.com", "notequal"}
	jid2 = &UnsafeJID{"newjid", "example.com", "notequal2"}
	if jid.Equal(jid2) {
		t.FailNow()
	}
}

// &UnsafeJIDs should be marshalable to an XML attribute
func TestMarshal(t *testing.T) {
	jid := &UnsafeJID{"newjid", "example.com", "marshal"}
	attr, err := jid.MarshalXMLAttr(xml.Name{Space: "", Local: "to"})

	if err != nil || attr.Name.Local != "to" || attr.Name.Space != "" || attr.Value != "newjid@example.com/marshal" {
		t.FailNow()
	}
}

// &UnsafeJIDs should be unmarshalable from an XML attribute
func TestUnmarshal(t *testing.T) {
	jid := &UnsafeJID{}
	err := jid.UnmarshalXMLAttr(
		xml.Attr{xml.Name{"space", ""}, "newjid@example.com/unmarshal"},
	)

	if err != nil || jid.Localpart() != "newjid" || jid.Domainpart() != "example.com" || jid.Resourcepart() != "unmarshal" {
		t.FailNow()
	}
}
