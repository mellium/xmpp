// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

const (
	// NS is the message styling namespace, exported as a convenience.
	NS = "urn:xmpp:styling:0"
)

// Unstyled is a type that can be added to messages to add a hint that will
// disable styling.
// When unmarshaled or marshaled its value indicates whether the unstyled hint
// was or will be present in the message.
type Unstyled struct {
	XMLName xml.Name `xml:"urn:xmpp:styling:0 unstyled"`
	Value   bool
}

// TokenReader implements xmlstream.Marshaler.
func (u Unstyled) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "unstyled"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (u Unstyled) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, u.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (u Unstyled) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := u.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (u *Unstyled) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	u.Value = start.Name.Space == NS && start.Name.Local == "unstyled"
	return d.Skip()
}

var (
	clientInserter = xmlstream.Insert(xml.Name{Space: stanza.NSClient, Local: "message"}, Unstyled{Value: true})
	serverInserter = xmlstream.Insert(xml.Name{Space: stanza.NSServer, Local: "message"}, Unstyled{Value: true})
)

// Disable is an xmlstream.Transformer that inserts a hint into any message read
// through r that disables styling for the body of the message.
func Disable(r xml.TokenReader) xml.TokenReader {
	return serverInserter(clientInserter(r))
}
