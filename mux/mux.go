// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mux implements an XMPP multiplexer.
package mux // import "mellium.im/xmpp/mux"

import (
	"encoding/xml"
	"fmt"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/iter"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/stanza"
)

const (
	iqStanza   = "iq"
	msgStanza  = "message"
	presStanza = "presence"
)

type pattern struct {
	Payload xml.Name
	Stanza  string
	Type    string
}

func (p pattern) String() string {
	return fmt.Sprintf("%s %s with payload {%s}%s", p.Type, p.Stanza, p.Payload.Space, p.Payload.Local)
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
	iqPatterns       map[pattern]IQHandler
	msgPatterns      map[pattern]MessageHandler
	presencePatterns map[pattern]PresenceHandler
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
		case iqStanza:
			return xmpp.HandlerFunc(m.iqRouter), true
		case msgStanza:
			return xmpp.HandlerFunc(m.msgRouter), true
		case presStanza:
			return xmpp.HandlerFunc(m.presenceRouter), true
		}
	}

	return nopHandler{}, false
}

// IQHandler returns the handler to use for an IQ payload with the given type
// and payload name.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) IQHandler(typ stanza.IQType, payload xml.Name) (h IQHandler, ok bool) {
	pattern := pattern{Stanza: iqStanza, Payload: payload, Type: string(typ)}
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

// MessageHandler returns the handler to use for a message with the given type
// and payload.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) MessageHandler(typ stanza.MessageType, payload xml.Name) (h MessageHandler, ok bool) {
	pattern := pattern{Stanza: msgStanza, Payload: payload, Type: string(typ)}
	h = m.msgPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = payload.Local
	h = m.msgPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = payload.Space
	pattern.Payload.Local = ""
	h = m.msgPatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = ""
	h = m.msgPatterns[pattern]
	if h != nil {
		return h, true
	}

	return nopHandler{}, false
}

// PresenceHandler returns the handler to use for a presence payload with the
// given type.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *ServeMux) PresenceHandler(typ stanza.PresenceType, payload xml.Name) (h PresenceHandler, ok bool) {
	pattern := pattern{Stanza: presStanza, Payload: payload, Type: string(typ)}
	h = m.presencePatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = payload.Local
	h = m.presencePatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = payload.Space
	pattern.Payload.Local = ""
	h = m.presencePatterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Payload.Space = ""
	pattern.Payload.Local = ""
	h = m.presencePatterns[pattern]
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
		pat := pattern{Stanza: iqStanza, Payload: payload, Type: string(typ)}
		if _, ok := m.iqPatterns[pat]; ok {
			panic("mux: multiple registrations for " + pat.String())
		}
		if m.iqPatterns == nil {
			m.iqPatterns = make(map[pattern]IQHandler)
		}
		m.iqPatterns[pat] = h
	}
}

// IQFunc returns an option that matches IQ stanzas.
// For more information see IQ.
func IQFunc(typ stanza.IQType, payload xml.Name, h IQHandler) Option {
	return IQ(typ, payload, h)
}

// Message returns an option that matches message stanzas by type.
func Message(typ stanza.MessageType, payload xml.Name, h MessageHandler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil message handler")
		}
		pat := pattern{Stanza: msgStanza, Payload: payload, Type: string(typ)}
		if _, ok := m.msgPatterns[pat]; ok {
			panic("mux: multiple registrations for " + pat.String())
		}
		if m.msgPatterns == nil {
			m.msgPatterns = make(map[pattern]MessageHandler)
		}
		m.msgPatterns[pat] = h
	}
}

// MessageFunc returns an option that matches message stanzas.
// For more information see Message.
func MessageFunc(typ stanza.MessageType, payload xml.Name, h MessageHandlerFunc) Option {
	return Message(typ, payload, h)
}

// Presence returns an option that matches presence stanzas by type.
func Presence(typ stanza.PresenceType, payload xml.Name, h PresenceHandler) Option {
	return func(m *ServeMux) {
		if h == nil {
			panic("mux: nil presence handler")
		}
		pat := pattern{Stanza: presStanza, Payload: payload, Type: string(typ)}
		if _, ok := m.presencePatterns[pat]; ok {
			panic("mux: multiple registrations for " + pat.String())
		}
		if m.presencePatterns == nil {
			m.presencePatterns = make(map[pattern]PresenceHandler)
		}
		m.presencePatterns[pat] = h
	}
}

// PresenceFunc returns an option that matches on presence stanzas.
// For more information see Presence.
func PresenceFunc(typ stanza.PresenceType, payload xml.Name, h PresenceHandlerFunc) Option {
	return Presence(typ, payload, h)
}

func isStanza(name xml.Name) bool {
	return (name.Local == iqStanza || name.Local == msgStanza || name.Local == presStanza) &&
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
	iq, err := stanza.NewIQ(*start)
	if err != nil {
		return err
	}

	// Limit the stream to the inside of the IQ element, don't allow handlers to
	// advance to the end token since they don't have access to the IQ start
	// token.
	t = struct {
		xml.TokenReader
		xmlstream.Encoder
	}{
		Encoder:     t,
		TokenReader: xmlstream.Inner(t),
	}
	tok, err := t.Token()
	if err != nil {
		return err
	}
	payloadStart, _ := tok.(xml.StartElement)
	h, _ := m.IQHandler(iq.Type, payloadStart.Name)
	return h.HandleIQ(iq, t, &payloadStart)
}

type bufReader struct {
	r      xml.TokenReader
	buf    []xml.Token
	offset uint
}

func (r *bufReader) Token() (xml.Token, error) {
	if r.offset < uint(len(r.buf)) {
		o := r.offset
		r.offset++
		return r.buf[o], nil
	}

	tok, err := r.r.Token()
	if tok != nil {
		tok = xml.CopyToken(tok)
		r.buf = append(r.buf, tok)
		r.offset++
	}
	return tok, err
}

// TODO: this is terrible error handling, figure out a better way to handle
// multiple errors that should be turned into a single stanza error.
type multiErr []error

func (e multiErr) Error() string {
	var buf strings.Builder
	for i, err := range e {
		if i == 0 {
			buf.WriteString(err.Error())
			continue
		}
		fmt.Fprintf(&buf, ", %s", err.Error())
	}
	return buf.String()
}

func (m *ServeMux) msgRouter(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	msg, err := stanza.NewMessage(*start)
	if err != nil {
		return err
	}

	return forChildren(m, msg, t, start)
}

func (m *ServeMux) presenceRouter(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	presence, err := stanza.NewPresence(*start)
	if err != nil {
		return err
	}

	return forChildren(m, presence, t, start)
}

func forChildren(m *ServeMux, stanzaVal interface{}, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	r := &bufReader{
		r: t,
		// TODO: figure out a good buffer size
		buf:    make([]xml.Token, 0, 10),
		offset: 1,
	}
	r.buf = append(r.buf, *start)

	// TODO: figure out a good buffer size
	errs := make([]error, 0, 10)
	iterator := iter.New(r)
	for iterator.Next() {
		start, _ := iterator.Current()

		var err error
		switch s := stanzaVal.(type) {
		case stanza.Presence:
			br := &bufReader{r: t, buf: r.buf}
			h, _ := m.PresenceHandler(s.Type, start.Name)
			err = h.HandlePresence(s, struct {
				xml.TokenReader
				xmlstream.Encoder
			}{
				TokenReader: br,
				Encoder:     t,
			})
			r.buf = br.buf
		case stanza.Message:
			br := &bufReader{r: t, buf: r.buf}
			h, _ := m.MessageHandler(s.Type, start.Name)
			err = h.HandleMessage(s, struct {
				xml.TokenReader
				xmlstream.Encoder
			}{
				TokenReader: br,
				Encoder:     t,
			})
			r.buf = br.buf
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if err := iterator.Err(); err != nil {
		return err
	}
	if len(errs) > 0 {
		return multiErr(errs)
	}
	// If the only tokens are the start and close tokens, trigger any wildcard
	// handlers.
	if len(r.buf) == 2 {
		r.offset = 0
		switch s := stanzaVal.(type) {
		case stanza.Presence:
			h, _ := m.PresenceHandler(s.Type, xml.Name{})
			return h.HandlePresence(s, struct {
				xml.TokenReader
				xmlstream.Encoder
			}{
				TokenReader: r,
				Encoder:     t,
			})
		case stanza.Message:
			h, _ := m.MessageHandler(s.Type, xml.Name{})
			return h.HandleMessage(s, struct {
				xml.TokenReader
				xmlstream.Encoder
			}{
				TokenReader: r,
				Encoder:     t,
			})
		}
	}
	return nil
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
	_, err := xmlstream.Copy(t, iq.Wrap(e.TokenReader()))
	return err
}
