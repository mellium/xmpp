// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

// Presence is an XMPP stanza that is used as an indication that an entity is
// available for communication. It is used to set a status message, broadcast
// availability, and advertise entity capabilities. It can be directed
// (one-to-one), or used as a broadcast mechanism (one-to-many).
type Presence struct {
	XMLName xml.Name     `xml:"presence"`
	ID      string       `xml:"id,attr"`
	To      jid.JID      `xml:"to,attr"`
	From    jid.JID      `xml:"from,attr"`
	Lang    string       `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    PresenceType `xml:"type,attr,omitempty"`
}

// NewPresence unmarshals an XML token into a Presence.
func NewPresence(start xml.StartElement) (Presence, error) {
	v := Presence{}
	d := xml.NewTokenDecoder(xmlstream.Wrap(nil, start))
	err := d.Decode(&v)
	return v, err
}

// StartElement converts the Presence into an XML token.
func (p Presence) StartElement() xml.StartElement {
	// Keep whatever namespace we're already using but make sure the localname is
	// "presence".
	name := p.XMLName
	name.Local = "presence"

	attr := make([]xml.Attr, 0, 5)
	if p.ID != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: p.ID})
	}
	if !p.To.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: p.To.String()})
	}
	if !p.From.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "from"}, Value: p.From.String()})
	}
	if p.Lang != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: p.Lang})
	}
	if p.Type != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: string(p.Type)})
	}

	return xml.StartElement{
		Name: name,
		Attr: attr,
	}
}

// Wrap wraps the payload in a stanza.
//
// If to is the zero value for jid.JID, no to attribute is set on the resulting
// presence.
func (p Presence) Wrap(payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, p.StartElement())
}

// PresenceType is the type of a presence stanza.
// It should normally be one of the constants defined in this package.
type PresenceType string

const (
	// AvailablePresence is a special case that signals that the entity is
	// available for communication.
	AvailablePresence PresenceType = ""

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
