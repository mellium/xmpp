// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"context"
	"encoding/xml"
	"testing"
)

var (
	boshLink = Link{Rel: "urn:xmpp:alt-connections:xbosh", Href: "https://web.example.com:5280/bosh"}
	wsLink   = Link{Rel: "urn:xmpp:alt-connections:websocket", Href: "wss://web.example.com:443/ws"}
)

func TestUnmarshalWellKnownXML(t *testing.T) {
	hostMeta := []byte(`<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
  <Link rel="urn:xmpp:alt-connections:xbosh"
        href="https://web.example.com:5280/bosh" />
  <Link rel="urn:xmpp:alt-connections:websocket"
        href="wss://web.example.com:443/ws" />
</XRD>`)
	var xrd XRD
	if err := xml.Unmarshal(hostMeta, &xrd); err != nil {
		t.Error(err)
	}
	switch {
	case len(xrd.Links) != 2:
		t.Errorf("Expected 2 links in xrd unmarshal output, but found %d", len(xrd.Links))
	case xrd.Links[0] != boshLink:
		t.Errorf("Expected %v, but got %v", boshLink, xrd.Links[0])
	case xrd.Links[1] != wsLink:
		t.Errorf("Expected %v, but got %v", wsLink, xrd.Links[1])
	}
}

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
