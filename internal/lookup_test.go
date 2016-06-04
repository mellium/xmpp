// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"encoding/json"
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
		t.Fatal(err)
	}
	switch {
	case len(xrd.Links) != 2:
		t.Fatalf("Expected 2 links in xrd unmarshal output, but found %d", len(xrd.Links))
	case xrd.Links[0] != boshLink:
		t.Fatalf("Expected %v, but got %v", boshLink, xrd.Links[0])
	case xrd.Links[1] != wsLink:
		t.Fatalf("Expected %v, but got %v", wsLink, xrd.Links[1])
	}
}

func TestUnmarshalWellKnownJSON(t *testing.T) {
	hostMeta := []byte(`{
  "links": [
    {
      "rel": "urn:xmpp:alt-connections:xbosh",
      "href": "https://web.example.com:5280/bosh"
    },
    {
      "rel": "urn:xmpp:alt-connections:websocket",
      "href": "wss://web.example.com:443/ws"
    }
  ]
}`)
	var xrd XRD
	if err := json.Unmarshal(hostMeta, &xrd); err != nil {
		t.Fatal(err)
	}
	switch {
	case len(xrd.Links) != 2:
		t.Fatalf("Expected 2 links in xrd unmarshal output, but found %d", len(xrd.Links))
	case xrd.Links[0] != boshLink:
		t.Fatalf("Expected %v, but got %v", boshLink, xrd.Links[0])
	case xrd.Links[1] != wsLink:
		t.Fatalf("Expected %v, but got %v", wsLink, xrd.Links[1])
	}
}
