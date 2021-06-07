// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature

// Package forward implements forwarding messages.
package forward // import "mellium.im/xmpp/forward"

import (
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package, provided as a convenience.
const (
	NS = "urn:xmpp:forward:0"
)

// Forwarded can be embedded into another struct along with a stanza to wrap the
// stanza for forwarding.
type Forwarded struct {
	XMLName xml.Name    `xml:"urn:xmpp:forward:0 forwarded"`
	Delay   delay.Delay `xml:"urn:xmpp:delay delay"`
}

// Wrap wraps the provided token reader (which should be a stanza, but this is
// not enforced) to prepare it for forwarding.
func (f Forwarded) Wrap(r xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.MultiReader(
			f.Delay.TokenReader(),
			r,
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "forwarded"}},
	)
}

// TokenReader implements xmlstream.Marshaler.
func (f Forwarded) TokenReader() xml.TokenReader {
	return f.Wrap(nil)
}

// WriteXML implements xmlstream.WriterTo.
func (f Forwarded) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// Wrap forwards the provided token stream by wrapping it in a new message
// stanza and recording the original delivery time of the stanza.
// The body is in addition to the forwarded stanza and is not meant as a
// fallback in case the forwarded message cannot be displayed.
//
// The token stream is expected to be a stanza, but this is not enforced.
func Wrap(msg stanza.Message, body string, received time.Time, r xml.TokenReader) xml.TokenReader {
	return msg.Wrap(xmlstream.MultiReader(
		xmlstream.Wrap(xmlstream.Token(xml.CharData(body)), xml.StartElement{Name: xml.Name{Local: "body"}}),
		Forwarded{
			Delay: delay.Delay{
				Time: received,
			},
		}.Wrap(r),
	))
}
