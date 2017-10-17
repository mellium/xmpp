// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"testing"

	"mellium.im/xmpp/jid"
)

func TestDialClientPanicsIfNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Dial to panic when passed a nil context.")
		}
	}()
	DialClient(nil, "tcp", jid.MustParse("feste@shakespeare.lit"))
}

// The default value of config.conntype should return "xmpp-client"
func TestDefaultConnType(t *testing.T) {
	if ct := connType(false); ct != "xmpp-client" {
		t.Errorf("Wrong default value for conntype; expected xmpp-client but got %s", ct)
	}
}

// If S2S is true, config.conntype should return "xmpp-server"
func TestS2SConnType(t *testing.T) {
	if ct := connType(true); ct != "xmpp-server" {
		t.Errorf("Wrong s2s value for conntype; expected xmpp-server but got %s", ct)
	}
}
