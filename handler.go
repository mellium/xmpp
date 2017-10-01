// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"fmt"
	"sync"
)

// A Handler triggers events or responds to incoming elements in an XML stream.
type Handler interface {
	HandleXMPP(t xml.TokenReader, start *xml.StartElement) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// XMPP handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(t xml.TokenReader, start *xml.StartElement) error

// HandleXMPP calls f(t, start).
func (f HandlerFunc) HandleXMPP(t xml.TokenReader, start *xml.StartElement) error {
	return f(t, start)
}

// ServeMux is an XMPP request multiplexer. It matches the local name and
// namespace of each top level start element token against a list of registered
// handlers and calls the handler for the pattern that most closely matches the
// namespace.
//
// BE ADVISED: This API will almost certainly change radically before 1.0 or be
// removed entirely. Don't use it.
type ServeMux struct {
	mu sync.RWMutex
	m  map[string]muxEntry
}

type muxEntry struct {
	h       Handler
	pattern string
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if pattern == "" {
		panic("xmpp: invalid empty pattern")
	}
	if handler == nil {
		panic("xmpp: nil handler")
	}
	if _, ok := mux.m[pattern]; ok {
		panic("xmpp: multiple registrations for " + pattern)
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	mux.m[pattern] = muxEntry{h: handler, pattern: pattern}
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(xml.TokenReader, *xml.StartElement) error) {
	mux.Handle(pattern, HandlerFunc(handler))
}

// Handler returns the handler to use for the given request, consulting name.
// It always returns a non-nil handler.
func (mux *ServeMux) Handler(name xml.Name) (h Handler, pattern string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	// Check for exact match first.
	v, ok := mux.m[fmt.Sprintf("%s %s", name.Space, name.Local)]
	if ok {
		return v.h, v.pattern
	}

	panic("xmpp: default handler not yet implemented")
}

func (mux *ServeMux) HandleXMPP(t xml.TokenReader, start *xml.StartElement) error {
	h, _ := mux.Handler(start.Name)
	return h.HandleXMPP(t, start)
}
