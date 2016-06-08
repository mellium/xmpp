// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"
)

// If an invalid connection type is looked up, we should panic.
func TestLookupEndpointPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupEndpoint should panic if an invalid conntype is specified.")
		}
	}()
	lookupEndpoint(nil, nil, nil, "wssorbashorsomething")
}

// If an invalid connection type is looked up, we should panic.
func TestLookupDNSPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupDNS should panic if an invalid conntype is specified.")
		}
	}()
	lookupDNS(nil, "name", "wssorbashorsomething")
}

// If an invalid connection type is looked up, we should panic.
func TestLookupHostMetaPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupHostMeta should panic if an invalid conntype is specified.")
		}
	}()
	lookupHostMeta(nil, nil, "name", "wssorbashorsomething")
}
