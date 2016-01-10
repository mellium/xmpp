// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// Compile time check ot make sure that JID and *JID match several interfaces.
var _ fmt.Stringer = (*JID)(nil)
var _ fmt.Stringer = JID{}
var _ xml.MarshalerAttr = (*JID)(nil)
var _ xml.MarshalerAttr = JID{}
var _ xml.UnmarshalerAttr = (*JID)(nil)

var invalid = string([]byte{0xff, 0xfe, 0xfd})

// JID's cannot contain invalid UTF8 in the localpart.
func TestNewInvalidUtf8Localpart(t *testing.T) {
	_, err := ParseString(invalid + "@example.com/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// JID's cannot contain invalid UTF8 in the domainpart.
func TestNewInvalidUtf8Domainpart(t *testing.T) {
	_, err := ParseString("example@" + invalid + "/resourcepart")
	if err == nil {
		t.FailNow()
	}
}

// JID's cannot contain invalid UTF8 in the resourcepart.
func TestNewInvalidUtf8Resourcepart(t *testing.T) {
	_, err := ParseString("example@example.com/" + invalid)
	if err == nil {
		t.FailNow()
	}
}
