// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.
package jid

import (
	"encoding/xml"
	"testing"
)

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

// New JIDs should not allow `\t` in the localpart.
func TestNewHasTabInLocalpart(t *testing.T) {
	_, err := FromString("localpart	@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// New JIDs should not allow spaces in the domainpart.
func TestNewHasSpaceInDomainpart(t *testing.T) {
	_, err := FromString("localpart@exampl e.com/resourcepart")
	if err == nil {
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
	jid := &Jid{"newjid", "example.com", "equal"}
	jid2 := &Jid{"newjid", "example.com", "equal"}
	if !jid.Equals(jid2) {
		t.FailNow()
	}
}

// Two different JIDs should not be equal.
func TestNotEqualJIDs(t *testing.T) {
	jid := &Jid{"newjid", "example.com", "notequal"}
	jid2 := &Jid{"newjid2", "example.com", "notequal"}
	if jid.Equals(jid2) {
		t.FailNow()
	}
	jid = &Jid{"newjid", "example.com", "notequal"}
	jid2 = &Jid{"newjid", "example.net", "notequal"}
	if jid.Equals(jid2) {
		t.FailNow()
	}
	jid = &Jid{"newjid", "example.com", "notequal"}
	jid2 = &Jid{"newjid", "example.com", "notequal2"}
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
	jid := Jid{"newjid", "example.com", "marshal"}
	attr, err := jid.MarshalXMLAttr(xml.Name{Space: "", Local: "to"})

	if err != nil || attr.Name.Local != "to" || attr.Name.Space != "" || attr.Value != "newjid@example.com/marshal" {
		t.FailNow()
	}
}
