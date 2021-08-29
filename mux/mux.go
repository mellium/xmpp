// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mux contains a simple XMPP multiplexer.
//
// Aside from implementing its own muxer, this package contains handler
// interfaces designed to be standard across multiplexers.
// This lets you write, for example, a muxer that matches elements based on
// xpath expressions and take advantage of existing handlers.
package mux // import "mellium.im/xmpp/mux"

import (
	"encoding/xml"
	"fmt"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/form"
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

// ForItems implements items.Iter for the mux by iterating over all child items.
func (m *ServeMux) ForItems(node string, f func(items.Item) error) error {
	for _, h := range m.patterns {
		if itemIter, ok := h.(items.Iter); ok {
			err := itemIter.ForItems(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.iqPatterns {
		if itemIter, ok := h.(items.Iter); ok {
			err := itemIter.ForItems(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.msgPatterns {
		if itemIter, ok := h.(items.Iter); ok {
			err := itemIter.ForItems(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.presencePatterns {
		if itemIter, ok := h.(items.Iter); ok {
			err := itemIter.ForItems(node, f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ForFeatures implements info.FeatureIter for the mux by iterating over all
// child features.
func (m *ServeMux) ForFeatures(node string, f func(info.Feature) error) error {
	for _, h := range m.patterns {
		if featureIter, ok := h.(info.FeatureIter); ok {
			err := featureIter.ForFeatures(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.iqPatterns {
		if featureIter, ok := h.(info.FeatureIter); ok {
			err := featureIter.ForFeatures(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.msgPatterns {
		if featureIter, ok := h.(info.FeatureIter); ok {
			err := featureIter.ForFeatures(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.presencePatterns {
		if featureIter, ok := h.(info.FeatureIter); ok {
			err := featureIter.ForFeatures(node, f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ForIdentities implements info.IdentityIter for the mux by iterating over
// all child handlers.
func (m *ServeMux) ForIdentities(node string, f func(info.Identity) error) error {
	for _, h := range m.patterns {
		if identIter, ok := h.(info.IdentityIter); ok {
			err := identIter.ForIdentities(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.iqPatterns {
		if identIter, ok := h.(info.IdentityIter); ok {
			err := identIter.ForIdentities(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.msgPatterns {
		if identIter, ok := h.(info.IdentityIter); ok {
			err := identIter.ForIdentities(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.presencePatterns {
		if identIter, ok := h.(info.IdentityIter); ok {
			err := identIter.ForIdentities(node, f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ForIdentities implements form.Iter for the mux by iterating over all child
// handlers.
func (m *ServeMux) ForForms(node string, f func(*form.Data) error) error {
	for _, h := range m.patterns {
		if formIter, ok := h.(form.Iter); ok {
			err := formIter.ForForms(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.iqPatterns {
		if formIter, ok := h.(form.Iter); ok {
			err := formIter.ForForms(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.msgPatterns {
		if formIter, ok := h.(form.Iter); ok {
			err := formIter.ForForms(node, f)
			if err != nil {
				return err
			}
		}
	}
	for _, h := range m.presencePatterns {
		if formIter, ok := h.(form.Iter); ok {
			err := formIter.ForForms(node, f)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

	iterator := xmlstream.NewIter(r)
	/* #nosec */
	defer iterator.Close()

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
