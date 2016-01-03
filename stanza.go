// Copyright 2015 Sam Whited.
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

// Message is an XMPP stanza that contains a payload for direct one-to-one
// communication with another network entity.  It is often used for sending chat
// messages to an individual or group chat server, or for notifications and
// alerts that don't require a response.
type Message struct {
	stanza
	XMLName xml.Name `xml:"message"`
}

// Presence is an XMPP stanza that is used as an indication that an entity is
// available for communication. It is used to set a status message, broadcast
// availability, and advertise entity capabilities. It can be directed
// (one-to-one), or as a broadcast mechanism (one-to-many).
type Presence struct {
	stanza
	XMLName xml.Name `xml:"presence"`
}

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	stanza
	XMLName xml.Name `xml:"iq"`
}
