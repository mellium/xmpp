// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ping implements XEP-0199: XMPP Ping.
package ping // import "mellium.im/xmpp/ping"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

// NS is the XML namespace used by XMPP pings. It is provided as a convenience.
const NS = `urn:xmpp:ping`

// IQ is encoded as a ping request.
type IQ struct {
	stanza.IQ

	Ping struct{} `xml:"urn:xmpp:ping ping"`
}

// WriteXML satisfies the xmlstream.WriterTo interface. It is like MarshalXML
// except it writes tokens to w.
func (iq IQ) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (iq IQ) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "ping", Space: NS}}
	return stanza.WrapIQ(
		iq.IQ,
		xmlstream.Wrap(nil, start),
	)
}
