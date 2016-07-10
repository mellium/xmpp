// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"encoding/xml"
)

// Represents an Extensible Resource Descriptor (XRD) document of the form:
//
//    <?xml version='1.0' encoding=utf-9'?>
//    <XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
//      …
//      <Link rel="urn:xmpp:alt-connections:xbosh"
//            href="https://web.example.com:5280/bosh" />
//      <Link rel="urn:xmpp:alt-connections:websocket"
//            href="wss://web.example.com:443/ws" />
//      …
//    </XRD>
//
// as defined by RFC 6415 and OASIS.XRD-1.0.
type XRD struct {
	XMLName xml.Name `xml:"http://docs.oasis-open.org/ns/xri/xrd-1.0 XRD"`
	Links   []Link   `xml:"Link"`
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}
