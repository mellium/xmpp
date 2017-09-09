// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

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
	To      *jid.JID     `xml:"to,attr"`
	From    *jid.JID     `xml:"from,attr"`
	Lang    string       `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    PresenceType `xml:"type,attr,omitempty"`
}

// IQType is the type of a presence stanza.
// It should normally be one of the constants defined in this package.
type PresenceType string

const (
	// ErrorPresence indicates that an error has occurred regarding processing of
	// a previously sent presence stanza; if the presence stanza is of type
	// "error", it MUST include an <error/> child element
	ErrorPresence PresenceType = "error"

	// ProbePresence is a request for an entity's current presence. It should
	// generally only be generated and sent by servers on behalf of a user.
	ProbePresence PresenceType = "probe"

	// SubscribePresence is sent when the sender wishes to subscribe to the
	// recipient's presence.
	SubscribePresence PresenceType = "subscribe"

	// SubscribedPresence indicates that the sender has allowed the recipient to
	// receive future presence broadcasts.
	SubscribedPresence PresenceType = "subscribed"

	// UnavailablePresence indicates that the sender is no longer available for
	// communication.
	UnavailablePresence PresenceType = "unavailable"

	// UnsubscribePresence indicates that the sender is unsubscribing from the
	// receiver's presence.
	UnsubscribePresence PresenceType = "unsubscribe"

	// UnsubscribedPresence indicates that the subscription request has been
	// denied, or a previously granted subscription has been revoked.
	UnsubscribedPresence PresenceType = "unsubscribed"
)
