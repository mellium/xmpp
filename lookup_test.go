// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"

	"golang.org/x/net/context"
)

// If an invalid connection type is looked up, we should panic.
func TestLookupEndpointPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupEndpoint should panic if an invalid conntype is specified.")
		}
	}()
	lookupEndpoint(context.Background(), nil, nil, "wssorbashorsomething")
}

// If an invalid connection type is looked up, we should panic.
func TestLookupDNSPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupDNS should panic if an invalid conntype is specified.")
		}
	}()
	lookupDNS(context.Background(), "name", "wssorbashorsomething")
}

// If an invalid connection type is looked up, we should panic.
func TestLookupHostMetaPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupHostMeta should panic if an invalid conntype is specified.")
		}
	}()
	lookupHostMeta(context.Background(), nil, "name", "wssorbashorsomething")
}

// The lookup methods should not run if the context is canceled.
func TestLookupMethodsDoNotRunIfContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := lookupDNS(ctx, "name", "ws"); err != context.Canceled {
		t.Error("lookupDNS should not run if the context is canceled.")
	}
	if _, err := lookupHostMeta(ctx, nil, "name", "ws"); err != context.Canceled {
		t.Error("lookupHostMeta should not run if the context is canceled.")
	}
}
