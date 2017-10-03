// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

// A Handler triggers events or responds to incoming elements in an XML stream.
type Handler interface {
	HandleXMPP(t xmlstream.TokenReadWriter, start *xml.StartElement) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// XMPP handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(t xmlstream.TokenReadWriter, start *xml.StartElement) error

// HandleXMPP calls f(t, start).
func (f HandlerFunc) HandleXMPP(t xmlstream.TokenReadWriter, start *xml.StartElement) error {
	return f(t, start)
}
