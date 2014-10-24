// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import "testing"

// Trying to create a new JID with an invalid UTF8 string should fail.
func TestNewInvalidUtf8Jid(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := NewJID(invalid + "@example.com/resourcepart")
	if err == nil || err.Error() != ERROR_INVALID_STRING {
		t.FailNow()
	}
}

// Trying to create a new bare JID (no resource part) should error.
func TestNewMissingResourcePart(t *testing.T) {
	_, err := NewJID("barejid@example.com")
	if err == nil || err.Error() != ERROR_NO_RESOURCE {
		t.FailNow()
	}
}

// Trying to create a JID with no localpart should error.
func TestNewMissingLocalPart(t *testing.T) {
	_, err := NewJID("@example.com/resourcepart")
	if err == nil || err.Error() != ERROR_INVALID_JID {
		t.FailNow()
	}
}

// Trying to create a JID with no @ symbol should error.
func TestNewMissingAtSymbol(t *testing.T) {
	_, err := NewJID("example.com/resourcepart")
	if err == nil || err.Error() != ERROR_INVALID_JID {
		t.FailNow()
	}
}

// New JIDs should strip whitespace from inputs.
func TestNewSurroundingWhitespace(t *testing.T) {
	jid, err := NewJID("  localpart@example.com/resourcepart	 ")
	if err != nil || jid.String() != "localpart@example.com/resourcepart" {
		t.FailNow()
	}
}

// New JIDs should not allow `\t`.
func TestNewHasTab(t *testing.T) {
	_, err := NewJID("localpart	@example.com/resourcepart")
	if err == nil || err.Error() != ERROR_ILLEGAL_SPACE {
		t.FailNow()
	}
}

// New JIDs should not allow spaces.
func TestNewHasSpace(t *testing.T) {
	_, err := NewJID("localpart@exampl e.com/resourcepart")
	if err == nil || err.Error() != ERROR_ILLEGAL_SPACE {
		t.FailNow()
	}
}

// Creating a new JID from a valid JID string should work and contain all the
// correct parts.
func TestNewValid(t *testing.T) {
	s := "jid@example.com/resourcepart"
	jid, err := NewJID(s)
	switch {
	case err != nil:
		t.FailNow();
	case jid.LocalPart() != "jid":
		t.FailNow();
	case jid.DomainPart() != "example.com":
		t.FailNow();
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
