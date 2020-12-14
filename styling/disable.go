// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling

import (
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
)

const (
	// NS is the message styling namespace, exported as a convenience.
	NS = "urn:xmpp:styling:0"
)

// Unstyled is a type that can be embedded in messages to add a hint that will
// disable styling.
// When unmarshaled its value indicates whether the unstyled hint was present in
// the message.
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
	if err != nil {
		return err
	}
	return e.Flush()
}

// UnmarshalXML implements xml.Unmarshaler.
func (u *Unstyled) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	u.Value = start.Name.Space == NS && start.Name.Local == "unstyled"
	return d.Skip()
}

// Disable inserts a hint into any message read through r that disables styling
// for the body of the message.
func Disable(r xml.TokenReader) xml.TokenReader {
	var inner xml.TokenReader
	return xmlstream.ReaderFunc(func() (xml.Token, error) {
		if inner != nil {
			tok, err := inner.Token()
			switch {
			case tok != nil && err == io.EOF:
				inner = nil
				return tok, nil
				// We don't need this case here because the Wrap/Token calls in the
				// multireader are optimized for early EOF and the multireader respects
				// that.
				// case tok == nil && err == io.EOF:
				//	inner = nil
			default:
				return tok, err
			}
		}

		tok, err := r.Token()
		if err != nil {
			return tok, err
		}

		if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "message" && (end.Name.Space == ns.Client || end.Name.Space == ns.Server) {
			inner = xmlstream.MultiReader(
				xmlstream.Wrap(nil,
					xml.StartElement{
						Name: xml.Name{Space: NS, Local: "unstyled"},
					},
				),
				xmlstream.Token(end),
			)
			return inner.Token()
		}

		return tok, err
	})
}
