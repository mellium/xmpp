// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"
)

// TODO: Make this a table test and add some more complicated messages.
// TODO: How should we test marshalling? Probably don't want to assume that
//       attribute order will remain stable.

func TestDefaults(t *testing.T) {
	var mt messageType

	if mt != NormalMessage {
		t.Log("Default value of message type should be 'normal'.")
		t.Fail()
	}
}
