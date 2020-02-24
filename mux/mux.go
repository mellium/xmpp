// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mux implements an XMPP multiplexer.
package mux // import "mellium.im/xmpp/mux"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type iqPattern struct {
	Payload xml.Name
	Type    stanza.IQType
}

// ServeMux is an XMPP stream multiplexer.
// It matches the start element token of each top level stream element against a
// list of registered patterns and calls the handler for the pattern that most
// closely matches the token.
//
// Patterns are XML names.
// If either the namespace or the localname is left off, any namespace or
// localname will be matched.
// Full XML names take precedence, followed by wildcard localnames, followed by
// wildcard namespaces.
type ServeMux struct {
	patterns         map[xml.Name]xmpp.Handler
	iqPatterns       map[iqPattern]IQHandler
	msgPatterns      map[stanza.MessageType]MessageHandler
	presencePatterns map[stanza.PresenceType]PresenceHandler
}

// New allocates and returns a new ServeMux.
func New(opt ...Option) *ServeMux {
	m := &ServeMux{}
	for _, o := range opt {
		o(m)
	}
	return m
}

// Handler returns the handler to use for a top level element with the provided
// XML name.
// If no exact match or wildcard handler exists, a default handler is returned
// (h is always non-nil) and ok will be false.
func (m *ServeMux) Handler(name xml.Name) (h xmpp.Handler, ok bool) {
	h = m.patterns[name]
	if h != nil {
		return h, true
	}

	n := name
	n.Space = ""
	h = m.patterns[n]
	if h != nil {
		return h, true
	}

	n = name
	n.Local = ""
	h = m.patterns[n]
	if h != nil {
		return h, true
	}

	if name.Space == ns.Client || name.Space == ns.Server {
		switch name.Local {
		case "iq":
			return xmpp.HandlerFunc(m.iqRouter), true
		case "message":
			return xmpp.HandlerFunc(m.msgRouter), true
		case "presence":
			return xmpp.HandlerFunc(m.presenceRouter), true
		}
	}

	return nopHandler{}, false
}

// IQHandler returns the handler to use for an IQ payload with the given type
// and payload name.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) IQHandler(typ stanza.IQType, payload xml.Name) (h IQHandler, ok bool) {
	pattern := iqPattern{Payload: payload, Type: typ}
	h = m.iqPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = payload.Local
	h = m.iqPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = payload.Space
	pattern.Payload.Local = ""
	h = m.iqPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = ""
	h = m.iqPatterns[pattern]
	if h != nil {
		return h, true
	}

	return IQHandlerFunc(iqFallback), false
}

// MessageHandler returns the handler to use for a message payload with the
// given type.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) MessageHandler(typ stanza.MessageType) (h MessageHandler, ok bool) {
	h = m.msgPatterns[typ]
	if h != nil {
		return h, true
	}

	return nopHandler{}, false
}

// PresenceHandler returns the handler to use for a presence payload with the
// given type.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) PresenceHandler(typ stanza.PresenceType) (h PresenceHandler, ok bool) {
	h = m.presencePatterns[typ]
	if h != nil {
		return h, true
	}

	return nopHandler{}, false
}

// HandleXMPP dispatches the request to the handler that most closely matches.
func (m *ServeMux) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h, _ := m.Handler(start.Name)
	return h.HandleXMPP(t, start)
}

// Option configures a ServeMux.
type Option func(m *ServeMux)

// IQ returns an option that matches IQ stanzas based on their type and the name
// of the payload.
func IQ(typ stanza.IQType, payload xml.Name, h IQHandler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil IQ handler")
		}
		pattern := iqPattern{Payload: payload, Type: typ}
		if _, ok := m.iqPatterns[pattern]; ok {
			panic("mux: multiple registrations for " + string(typ) + " iq with payload {" + pattern.Payload.Space + "}" + pattern.Payload.Local)
		}
		if m.iqPatterns == nil {
			m.iqPatterns = make(map[iqPattern]IQHandler)
		}
		m.iqPatterns[pattern] = h
	}
}

// IQFunc returns an option that matches IQ stanzas.
// For more information see IQ.
func IQFunc(typ stanza.IQType, payload xml.Name, h IQHandler) Option {
	return IQ(typ, payload, h)
}

// Message returns an option that matches message stanzas by type.
func Message(typ stanza.MessageType, h MessageHandler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil message handler")
		}
		if _, ok := m.msgPatterns[typ]; ok {
			panic("mux: multiple registrations for " + typ + " message")
		}
		if m.msgPatterns == nil {
			m.msgPatterns = make(map[stanza.MessageType]MessageHandler)
		}
		m.msgPatterns[typ] = h
	}
}

// MessageFunc returns an option that matches message stanzas.
// For more information see Message.
func MessageFunc(typ stanza.MessageType, h MessageHandlerFunc) Option {
	return Message(typ, h)
}

// Presence returns an option that matches presence stanzas by type.
func Presence(typ stanza.PresenceType, h PresenceHandler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil presence handler")
		}
		if _, ok := m.presencePatterns[typ]; ok {
			panic("mux: multiple registrations for " + typ + " presence")
		}
		if m.presencePatterns == nil {
			m.presencePatterns = make(map[stanza.PresenceType]PresenceHandler)
		}
		m.presencePatterns[typ] = h
	}
}

// PresenceFunc returns an option that matches on presence stanzas.
// For more information see Presence.
func PresenceFunc(typ stanza.PresenceType, h PresenceHandlerFunc) Option {
	return Presence(typ, h)
}

func isStanza(name xml.Name) bool {
	return (name.Local == "iq" || name.Local == "message" || name.Local == "presence") &&
		(name.Space == "" || name.Space == ns.Client || name.Space == ns.Server)
}

// Handle returns an option that matches on the provided XML name.
// If a handler already exists for n when the option is applied, the option
// panics.
func Handle(n xml.Name, h xmpp.Handler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil handler")
		}
		if isStanza(n) {
			panic("mux: tried to register stanza handler with Handle, use HandleIQ, HandleMessage, or HandlePresence instead")
		}
		if _, ok := m.patterns[n]; ok {
			panic("mux: multiple registrations for {" + n.Space + "}" + n.Local)
		}
		if m.patterns == nil {
			m.patterns = make(map[xml.Name]xmpp.Handler)
		}
		m.patterns[n] = h
	}
}

// HandleFunc returns an option that matches on the provided XML name.
func HandleFunc(n xml.Name, h xmpp.HandlerFunc) Option {
	return Handle(n, h)
}

type nopHandler struct{}

func (nopHandler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error { return nil }
func (nopHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error   { return nil }
func (nopHandler) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error   { return nil }

func (m *ServeMux) iqRouter(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	iq, err := newIQFromStart(start)
	if err != nil {
		return err
	}

	tok, err := t.Token()
	if err != nil {
		return err
	}
	payloadStart, _ := tok.(xml.StartElement)
	h, _ := m.IQHandler(iq.Type, payloadStart.Name)
	return h.HandleIQ(iq, t, &payloadStart)
}

func (m *ServeMux) msgRouter(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	msg, err := newMsgFromStart(start)
	if err != nil {
		return err
	}

	h, _ := m.MessageHandler(msg.Type)
	return h.HandleMessage(msg, t)
}

func (m *ServeMux) presenceRouter(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	presence, err := newPresenceFromStart(start)
	if err != nil {
		return err
	}

	h, _ := m.PresenceHandler(presence.Type)
	return h.HandlePresence(presence, t)
}

func iqFallback(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if iq.Type == stanza.ErrorIQ {
		return nil
	}

	iq.To, iq.From = iq.From, iq.To
	iq.Type = "error"

	e := stanza.Error{
		Type:      stanza.Cancel,
		Condition: stanza.ServiceUnavailable,
	}
	_, err := xmlstream.Copy(t, stanza.WrapIQ(
		iq,
		e.TokenReader(),
	))
	return err
}

// newIQFromStart takes a start element and returns an IQ.
func newIQFromStart(start *xml.StartElement) (stanza.IQ, error) {
	iq := stanza.IQ{}
	var err error
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			if a.Name.Space != "" {
				continue
			}
			iq.ID = a.Value
		case "to":
			if a.Name.Space != "" {
				continue
			}
			iq.To, err = jid.Parse(a.Value)
			if err != nil {
				return iq, err
			}
		case "from":
			if a.Name.Space != "" {
				continue
			}
			iq.From, err = jid.Parse(a.Value)
			if err != nil {
				return iq, err
			}
		case "lang":
			if a.Name.Space != ns.XML {
				continue
			}
			iq.Lang = a.Value
		case "type":
			if a.Name.Space != "" {
				continue
			}
			iq.Type = stanza.IQType(a.Value)
		}
	}
	return iq, nil
}

// newMsgFromStart takes a start element and returns a message.
func newMsgFromStart(start *xml.StartElement) (stanza.Message, error) {
	msg := stanza.Message{}
	var err error
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			if a.Name.Space != "" {
				continue
			}
			msg.ID = a.Value
		case "to":
			if a.Name.Space != "" {
				continue
			}
			msg.To, err = jid.Parse(a.Value)
			if err != nil {
				return msg, err
			}
		case "from":
			if a.Name.Space != "" {
				continue
			}
			msg.From, err = jid.Parse(a.Value)
			if err != nil {
				return msg, err
			}
		case "lang":
			if a.Name.Space != ns.XML {
				continue
			}
			msg.Lang = a.Value
		case "type":
			if a.Name.Space != "" {
				continue
			}
			msg.Type = stanza.MessageType(a.Value)
		}
	}
	return msg, nil
}

// newPresenceFromStart takes a start element and returns a message.
func newPresenceFromStart(start *xml.StartElement) (stanza.Presence, error) {
	presence := stanza.Presence{}
	var err error
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			if a.Name.Space != "" {
				continue
			}
			presence.ID = a.Value
		case "to":
			if a.Name.Space != "" {
				continue
			}
			presence.To, err = jid.Parse(a.Value)
			if err != nil {
				return presence, err
			}
		case "from":
			if a.Name.Space != "" {
				continue
			}
			presence.From, err = jid.Parse(a.Value)
			if err != nil {
				return presence, err
			}
		case "lang":
			if a.Name.Space != ns.XML {
				continue
			}
			presence.Lang = a.Value
		case "type":
			if a.Name.Space != "" {
				continue
			}
			presence.Type = stanza.PresenceType(a.Value)
		}
	}
	return presence, nil
}
