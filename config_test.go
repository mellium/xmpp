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
		t.Errorf("Wrong default value for conntype; expected xmpp-client but got %s", ct)
	}
}

// If S2S is true, config.conntype should return "xmpp-server"
func TestS2SConnType(t *testing.T) {
	c := &Config{S2S: true}
	if ct := c.connType(); ct != "xmpp-server" {
		t.Errorf("Wrong s2s value for conntype; expected xmpp-server but got %s", ct)
	}
}

// New configs should populate the features map with no duplicates.
func TestNewConfigShouldPopulateFeatures(t *testing.T) {
	c := NewServerConfig(nil, nil, BindResource(), BindResource(), StartTLS(true))
	if len(c.Features) != 2 {
		t.Errorf("Expected two features (Bind and StartTLS) but got: %v", c.Features)
	}
}
