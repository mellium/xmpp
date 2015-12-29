// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.
package jid

import (
	"testing"
)

// SafeJID's cannot contain invalid UTF8 in the localpart.
func TestNewInvalidUtf8Localpart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := SafeFromString(invalid + "@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// SafeJID's cannot contain invalid UTF8 in the domainpart.
func TestNewInvalidUtf8Domainpart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := SafeFromString("example@" + invalid + "/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// SafeJID's cannot contain invalid UTF8 in the resourcepart.
func TestNewInvalidUtf8Resourcepart(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_, err := SafeFromString("example@example.com/" + invalid)
	if err == nil {
		t.FailNow()
	}
}
