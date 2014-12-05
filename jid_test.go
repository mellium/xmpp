// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// Trying to create a new JID with an invalid UTF8 string should fail.
func TestNewInvalidUtf8Jid(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := NewJID(invalid + "@example.com/resourcepart")
	if err == nil || err.Error() != ErrorInvalidString {
		t.FailNow()
	}
}

// Trying to create a JID with an empty localpart should error.
func TestNewEmptyLocalPart(t *testing.T) {
	_, err := NewJID("@example.com/resourcepart")
	if err == nil || err.Error() != ErrorEmptyPart {
		t.FailNow()
	}
}

// Trying to create a JID with no localpart should work.
func TestNewNoLocalPart(t *testing.T) {
	jid, err := NewJID("example.com/resourcepart")
	if err != nil || jid.LocalPart() != "" {
		t.FailNow()
	}
}

// Trying to create a JID with no domainpart should error.
func TestNewNoDomainPart(t *testing.T) {
	_, err := NewJID("text@/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with no anything should error.
func TestNewNoAnything(t *testing.T) {
	_, err := NewJID("@/")
	if err == nil {
		t.FailNow()
	}
}

// Trying to create a JID with '@' or '/' in the resourcepart should work.
func TestNewJidInResourcePart(t *testing.T) {
	_, err := NewJID("this/is@/fine")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

// Trying to create a JID with an empty resource part should error.
func TestNewEmptyResourcePart(t *testing.T) {
	_, err := NewJID("text@example.com/")
	if err == nil || err.Error() != ErrorEmptyPart {
		t.FailNow()
	}
}

// Trying to create a new bare JID (no resource part) should work.
func TestNewBareJID(t *testing.T) {
	jid, err := NewJID("barejid@example.com")
	if err != nil || jid.ResourcePart() != "" {
		t.FailNow()
	}
}

// New JIDs should strip whitespace from inputs.
func TestNewSurroundingWhitespace(t *testing.T) {
	jid, err := NewJID("  localpart@example.com/resourcepart	 ")
	if err != nil {
		t.FailNow()
	}
	str := jid.String()
	if str != "localpart@example.com/resourcepart" {
		t.FailNow()
	}
}

// New JIDs should not allow `\t`.
func TestNewHasTab(t *testing.T) {
	_, err := NewJID("localpart	@example.com/resourcepart")
	if err == nil || err.Error() != ErrorIllegalSpace {
		t.FailNow()
	}
}

// New JIDs should not allow spaces.
func TestNewHasSpace(t *testing.T) {
	_, err := NewJID("localpart@exampl e.com/resourcepart")
	if err == nil || err.Error() != ErrorIllegalSpace {
		t.FailNow()
	}
}

// New JIDs should not be empty strings.
func TestNewEmpty(t *testing.T) {
	_, err := NewJID("")
	if err == nil {
		t.FailNow()
	}
}

// Creating a new JID from a valid JID string should work and contain all the
// correct parts.
func TestNewValid(t *testing.T) {
	s := "jid@example.com/resourcepart"
	jid, err := NewJID(s)
	if err != nil {
		t.FailNow()
	}
	dp, err := jid.DomainPart()
	switch {
	case err != nil:
		t.FailNow()
	case jid.LocalPart() != "jid":
		t.FailNow()
	case dp != "example.com":
		t.FailNow()
	case jid.ResourcePart() != "resourcepart":
		t.FailNow()
	}
}

// Two identical JIDs should be equal.
func TestEqualJIDs(t *testing.T) {
	jid := JID{"newjid", "example.com", "equal"}
	jid2 := JID{"newjid", "example.com", "equal"}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// Two different JIDs should not be equal.
func TestNotEqualJIDs(t *testing.T) {
	jid := JID{"newjid", "example.com", "notequal"}
	jid2 := JID{"newjid2", "example.com", "notequal"}
	if jid.Equals(jid2) {
		t.FailNow()
	}
}

// Two JIDs with similar looking unicode characters should be equal.
func TestEqualsUnicodeNorm(t *testing.T) {
	// U+2126 Ω ohm sign
	jid, err := NewJID("Ω@example.com/res")
	if err != nil {
		t.Fail()
	}
	// U+03A9 Ω greek capital letter omega
	jid2, err := NewJID("Ω@example.com/res")
	if err != nil {
		t.Fail()
	}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// Test marshalling JID into an XML attribute
func TestMarshal(t *testing.T) {
	jid := JID{"newjid", "example.com", "marshal"}
	attr, err := jid.MarshalXMLAttr(xml.Name{Space: "", Local: "to"})

	if err != nil || attr.Name.Local != "to" || attr.Name.Space != "" || attr.Value != "newjid@example.com/marshal" {
		t.FailNow()
	}
}
