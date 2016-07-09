// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"
)

func TestSASLPanicsNoMechanisms(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected call to SASL() with no mechanisms to panic")
		}
	}()
	_ = SASL()
}
