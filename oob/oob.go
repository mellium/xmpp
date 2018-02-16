// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package oob implements XEP-0066: Out of Band Data.
package oob // import "mellium.im/xmpp/oob"

import (
	"encoding/xml"

	"mellium.im/xmpp/stanza"
)

// OOB namespaces provided as a convenience.
const (
	NS      = `jabber:x:oob`
	NSQuery = `jabber:iq:oob`
)

// IQ represents an OOB data query; for instance:
//
//     <iq type='set'
//         from='feste@example.net/asegasd'
//         to='malvolio@jabber.org/apkjase'
//         id='asiepjg'>
//       <query xmlns='jabber:iq:oob'>
//         <url>https://xmpp.org/images/promo/xmpp_server_guide_2017.pdf</url>
//         <desc>XMPP Server Setup Guide 2017</desc>
//       </query>
//     </iq>
type IQ struct {
	stanza.IQ
	Query Query
}

// Query represents an OOB data node that might be placed in an IQ stanza.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:oob query"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
}

// Data represents an OOB data node that might be placed in a message or
// presence stanza.
type Data struct {
	XMLName xml.Name `xml:"jabber:x:oob x"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
}
