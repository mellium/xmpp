// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/jid"
)

// stanza contains fields common to any any top level XMPP stanza (Presence,
// Message, or IQ)
type stanza struct {
	ID    string  `xml:"id,attr"`
	Inner string  `xml:",innerxml"`
	To    jid.JID `xml:"to,attr"`
	From  jid.JID `xml:"from,attr"`
	Lang  string  `xml:"xml:lang,attr"`
}

// Message is a top level XMPP stanza that contains a payload for direct
// one-to-one communication with another network entity.  It is often used for
// sending chat messages to an individual or group chat server, or for
// notifications and alerts that don't require a response.
type Message struct {
	stanza
	XMLName xml.Name `xml:"message"`
}

// Presence rep
type Presence struct {
	stanza
	XMLName xml.Name `xml:"presence"`
}

type IQ struct {
	stanza
	XMLName xml.Name `xml:"iq"`
}
