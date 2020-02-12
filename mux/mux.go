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
	"mellium.im/xmpp/stanza"
)

// ServeMux is an XMPP stream multiplexer.
// It matches the start element token of each top level stream element against a
// list of registered patterns and calls the handler for the pattern that most
// closely matches the token.
//
// Patterns are XML names.
// If either the namespace or the localname is left off, any namespace or
// localname will be matched.
// Full XML names take precedence, followed by wildcard namespaces, followed by
// wildcard localnames.
type ServeMux struct {
	fallback xmpp.Handler
	patterns map[xml.Name]xmpp.Handler
}

func fallback(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if start.Name.Local != "iq" {
		return nil
	}

	typeIdx := -1
	toIdx := -1
	fromIdx := -1
	for i, a := range start.Attr {
		switch a.Name.Local {
		case "type":
			if a.Value == "error" {
				return nil
			}
			typeIdx = i
		case "from":
			fromIdx = i
		case "to":
			toIdx = i
		}
		if typeIdx > -1 && fromIdx > -1 && toIdx > -1 {
			break
		}
	}

	switch {
	case toIdx < 0 && fromIdx < 0:
		// nothing to do here
	case toIdx < 0:
		start.Attr[fromIdx].Name.Local = "to"
	case fromIdx < 0:
		start.Attr[toIdx].Name.Local = "from"
	default:
		// swap values
		start.Attr[toIdx].Value, start.Attr[fromIdx].Value = start.Attr[fromIdx].Value, start.Attr[toIdx].Value
	}

	// TODO: double check with RFC 6120 that if there is no type the default
	// is "get" (and thus an error response should be generated).
	if typeIdx >= 0 {
		start.Attr[typeIdx].Value = "error"
	} else {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "type"},
			Value: "error",
		})
	}

	e := stanza.Error{
		Type:      stanza.Cancel,
		Condition: stanza.FeatureNotImplemented,
	}
	_, err := xmlstream.Copy(t, xmlstream.Wrap(e.TokenReader(), *start))
	return err
}

// New allocates and returns a new ServeMux.
func New(opt ...Option) *ServeMux {
	m := &ServeMux{
		fallback: xmpp.HandlerFunc(fallback),
		patterns: make(map[xml.Name]xmpp.Handler),
	}
	for _, o := range opt {
		o(m)
	}
	return m
}

// Handler returns the handler to use for a top level element with the provided
// XML name.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) Handler(name xml.Name) (h xmpp.Handler, ok bool) {
	h = m.patterns[name]
	if h == nil {
		return m.fallback, false
	}
	return h, true
}

// HandleXMPP dispatches the request to the handler whose pattern most closely
// matches start.Name.
func (m *ServeMux) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h, ok := m.Handler(start.Name)
	if ok {
		return h.HandleXMPP(t, start)
	}

	n := start.Name
	n.Space = ""
	h, ok = m.Handler(n)
	if ok {
		return h.HandleXMPP(t, start)
	}

	n = start.Name
	n.Local = ""
	h, _ = m.Handler(n)
	return h.HandleXMPP(t, start)
}

// Option configures a ServeMux.
type Option func(m *ServeMux)

func registerStanza(local string, h xmpp.Handler) Option {
	return func(m *ServeMux) {
		if h == nil {
			return
		}
		n := xml.Name{Local: local, Space: ns.Client}
		m.patterns[n] = h
		n = xml.Name{Local: local, Space: ns.Server}
		m.patterns[n] = h
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
func Handle(n xml.Name, h xmpp.Handler) Option {
	return func(m *ServeMux) {
		m.patterns[n] = h
	}
}

// HandleFunc returns an option that matches on the provided XML name.
func HandleFunc(n xml.Name, h xmpp.HandlerFunc) Option {
	return func(m *ServeMux) {
		m.patterns[n] = h
	}
}
