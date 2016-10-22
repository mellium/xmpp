// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"mellium.im/xmpp/jid"
)

// Presence is an XMPP stanza that is used as an indication that an entity is
// available for communication. It is used to set a status message, broadcast
// availability, and advertise entity capabilities. It can be directed
// (one-to-one), or used as a broadcast mechanism (one-to-many).
type Presence struct {
	XMLName xml.Name     `xml:"presence"`
	ID      string       `xml:"id,attr"`
	Inner   string       `xml:",innerxml"`
	To      *jid.JID     `xml:"to,attr"`
	From    *jid.JID     `xml:"from,attr"`
	Lang    string       `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    presenceType `xml:"type,attr,omitempty"`
}

func (p Presence) copyPresence() Presence {
	return p
}

type presenceType int

const (
	// NoTypePresence is a special type that indicates that a stanza is a presence
	// stanza without a defined type (indicating availability on the network).
	NoTypePresence presenceType = iota

	// ErrorPresence indicates that an error has occurred regarding processing of
	// a previously sent presence stanza; if the presence stanza is of type
	// "error", it MUST include an <error/> child element
	ErrorPresence presenceType = iota

	// ProbePresence is a request for an entity's current presence. It should
	// generally only be generated and sent by servers on behalf of a user.
	ProbePresence

	// SubscribePresence is sent when the sender wishes to subscribe to the
	// recipient's presence.
	SubscribePresence

	// SubscribedPresence indicates that the sender has allowed the recipient to
	// receive future presence broadcasts.
	SubscribedPresence

	// UnavailablePresence indicates that the sender is no longer available for
	// communication.
	UnavailablePresence

	// UnsubscribePresence indicates that the sender is unsubscribing from the
	// receiver's presence.
	UnsubscribePresence

	// UnsubscribedPresence indicates that the subscription request has been
	// denied, or a previously granted subscription has been revoked.
	UnsubscribedPresence
)
