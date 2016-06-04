// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"mellium.im/xmpp/internal"
)

const (
	wsPrefix   = "_xmpp-client-websocket="
	boshPrefix = "_xmpp-client-xbosh="
	wsRel      = "urn:xmpp:alt-connections:websocket"
	boshRel    = "urn:xmpp:alt-connections:xbosh"
	hostMeta   = "/.well-known/host-meta"
)

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

// LookupWebsocket discovers websocket endpoints that are valid for the given
// address using DNS TXT records and Web Host Metadata as described in XEP-0156:
// Discovering Alternative XMPP Connection Methods.
// func LookupWebsocket(addr *jid.JID) {
// }

func lookupWebsocketDNS(name string) (urls []string, err error) {
	txts, err := net.LookupTXT(name)
	if err != nil {
		return urls, err
	}

	var s string
	for _, txt := range txts {
		if s = strings.TrimPrefix(txt, wsPrefix); s != txt {
			urls = append(urls, s)
		}
	}

	return urls, err
}

func lookupWebsocketHostMeta(name string) (urls []string, err error) {
	url, err := url.Parse(name)
	if err != nil {
		return urls, err
	}
	url.Path = hostMeta
	resp, err := http.Get(path.Join(name, hostMeta))
	if err != nil {
		return urls, err
	}
	d := xml.NewDecoder(resp.Body)
	t, err := d.Token()

	// Tokenize the response until we find the first <XRD/> element.
	var xrd internal.XRD
	for {
		if se, ok := t.(xml.StartElement); ok && se.Name == xrdName {
			if err = d.DecodeElement(&xrd, &se); err != nil {
				return urls, err
			}
			break
		}
	}

	for _, link := range xrd.Links {
		if link.Rel == wsRel {
			urls = append(urls, link.Href)
		}
	}
	return urls, err
}
