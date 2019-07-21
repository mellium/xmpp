// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

// A Handler triggers events or responds to incoming elements in an XML stream.
type Handler interface {
	HandleXMPP(s *Session, start *xml.StartElement) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// XMPP handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(s *Session, start *xml.StartElement) error

// HandleXMPP calls f(t, start).
func (f HandlerFunc) HandleXMPP(s *Session, start *xml.StartElement) error {
	return f(s, start)
}
