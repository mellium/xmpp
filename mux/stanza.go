// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mux

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

// IQHandler responds to IQ stanzas.
type IQHandler interface {
	HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error
}

// The IQHandlerFunc type is an adapter to allow the use of ordinary functions
// as IQ handlers.
// If f is a function with the appropriate signature, IQHandlerFunc(f) is an
// IQHandler that calls f.
type IQHandlerFunc func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error

// HandleIQ calls f(iq, t, start).
func (f IQHandlerFunc) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return f(iq, t, start)
}

// MessageHandler responds to message stanzas.
type MessageHandler interface {
	HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error
}

// The MessageHandlerFunc type is an adapter to allow the use of ordinary
// functions as message handlers.
// If f is a function with the appropriate signature, MessageHandlerFunc(f) is a
// MessageHandler that calls f.
type MessageHandlerFunc func(msg stanza.Message, t xmlstream.TokenReadEncoder) error

// HandleMessage calls f(msg, t).
func (f MessageHandlerFunc) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	return f(msg, t)
}

// PresenceHandler responds to message stanzas.
type PresenceHandler interface {
	HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error
}

// The PresenceHandlerFunc type is an adapter to allow the use of ordinary
// functions as presence handlers.
// If f is a function with the appropriate signature, PresenceHandlerFunc(f) is
// a PresenceHandler that calls f.
type PresenceHandlerFunc func(p stanza.Presence, t xmlstream.TokenReadEncoder) error

// HandlePresence calls f(p, t).
func (f PresenceHandlerFunc) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
	return f(p, t)
}
