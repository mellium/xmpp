// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"
)

// The default value of config.conntype should return "xmpp-client"
func TestDefaultConnType(t *testing.T) {
	c := &Config{}
	if ct := c.connType(); ct != "xmpp-client" {
		t.Error("Wrong default value for conntype; expected xmpp-client but got %s", ct)
	}
}

// If S2S is true, config.conntype should return "xmpp-server"
func TestS2SConnType(t *testing.T) {
	c := &Config{S2S: true}
	if ct := c.connType(); ct != "xmpp-server" {
		t.Error("Wrong s2s value for conntype; expected xmpp-server but got %s", ct)
	}
}
