// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"fmt"
	"testing"
)

// Trying to create a new JID with an invalid UTF8 string should fail.
func TestNewInvalidUtf8Jid(t *testing.T) {
	// TODO: Leave this here while development is ongoing to make Google shutup
	// about unused imports... ugly hax.
	_ = fmt.Println
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

// New JIDs should not allow space.
func TestNewHasWhitespace(t *testing.T) {
	_, err := NewJID("localpart	@example.com/resourcepart")
	if err == nil {
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
		fallthrough
	case jid.LocalPart() != "jid":
		fallthrough
	case jid.DomainPart() != "example.com":
		fallthrough
	case jid.ResourcePart() != "resourcepart":
		t.FailNow()
	}
}
