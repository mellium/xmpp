// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package websocket

import "encoding/xml"

type open struct {
	To      string   `xml:"to,attr,omitempty"`
	From    string   `xml:"from,attr,omitempty"`
	Version string   `xml:"version,attr,omitempty"`
	Lang    string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	ID      string   `xml:"id,attr,omitempty"`
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-framing open"`
}

type close struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-framing close"`
}
