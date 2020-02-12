// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mux implements an XMPP multiplexer.
package mux

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/ns"
)

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
	patterns map[xml.Name]xmpp.Handler
}

func fallback(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if start.Name.Local != "iq" {
		return nil
	}

	iq, start, err := getPayload(t, start)
	if err != nil {
		return err
	}

	return iqFallback(iq, t, start)
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

	return xmpp.HandlerFunc(fallback), false
}

// HandleXMPP dispatches the request to the handler whose pattern most closely
// matches start.Name.
func (m *ServeMux) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h, _ := m.Handler(start.Name)
	return h.HandleXMPP(t, start)
}

// Option configures a ServeMux.
type Option func(m *ServeMux)

func registerStanza(local string, h xmpp.Handler) Option {
	return func(m *ServeMux) {
		Handle(xml.Name{Local: local, Space: ns.Client}, h)(m)
		Handle(xml.Name{Local: local, Space: ns.Server}, h)(m)
	}
}

// IQ returns an option that matches on all IQ stanzas.
func IQ(h xmpp.Handler) Option {
	return registerStanza("iq", h)
}

// IQFunc returns an option that matches on all IQ stanzas.
func IQFunc(h xmpp.HandlerFunc) Option {
	return IQ(h)
}

// Message returns an option that matches on all message stanzas.
func Message(h xmpp.Handler) Option {
	return registerStanza("message", h)
}

// MessageFunc returns an option that matches on all message stanzas.
func MessageFunc(h xmpp.HandlerFunc) Option {
	return Message(h)
}

// Presence returns an option that matches on all presence stanzas.
func Presence(h xmpp.Handler) Option {
	return registerStanza("presence", h)
}

// PresenceFunc returns an option that matches on all presence stanzas.
func PresenceFunc(h xmpp.HandlerFunc) Option {
	return Presence(h)
}

// Handle returns an option that matches on the provided XML name.
// If a handler already exists for n when the option is applied, the option
// panics.
func Handle(n xml.Name, h xmpp.Handler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil handler")
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
