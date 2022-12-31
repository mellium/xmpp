// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package saslerr provides error conditions for the XMPP profile of SASL as
// defined by RFC 6120 §6.5.
package saslerr // import "mellium.im/xmpp/internal/saslerr"

//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -type=Condition -linecomment

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
)

// Condition represents a SASL error condition that can be encapsulated by a
// <failure/> element.
type Condition uint16

// Standard SASL error conditions.
const (
	// None is a special condition that is used only if a defined condition was
	// not present. Its use violates the spec.
	ConditionNone                 Condition = iota // none
	ConditionAborted                               // aborted
	ConditionAccountDisabled                       // account-disabled
	ConditionCredentialsExpired                    // credentials-expired
	ConditionEncryptionRequired                    // encryption-required
	ConditionIncorrectEncoding                     // incorrect-encoding
	ConditionInvalidAuthzID                        // invalid-authzid
	ConditionInvalidMechanism                      // invalid-mechanism
	ConditionMalformedRequest                      // malformed-request
	ConditionMechanismTooWeak                      // mechanism-too-weak
	ConditionNotAuthorized                         // not-authorized
	ConditionTemporaryAuthFailure                  // temporary-auth-failure
)

// TokenReader implements the xmlstream.Marshaler interface.
func (c Condition) TokenReader() xml.TokenReader {
	if c == ConditionNone || c >= Condition(len(_Condition_index)-1) {
		return xmlstream.Token(nil)
	}
	return xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Local: c.String()}},
	)
}

// WriteXML implements the xmlstream.WriterTo interface.
func (c Condition) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, c.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface for a Failure.
func (c Condition) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := c.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// UnmarshalXML satisfies the xml.Unmarshaler interface.
func (c *Condition) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for cond := Condition(1); cond < Condition(len(_Condition_index)-1); cond++ {
		if start.Name.Local == cond.String() {
			*c = cond
			break
		}
	}
	return d.Skip()
}

// Error represents a SASL error that is marshalable to XML.
type Error struct {
	Condition Condition
	Lang      string
	Text      string
}

// Error satisfies the error interface for a Failure. It returns the text string
// if set, or the condition otherwise.
func (f Error) Error() string {
	if f.Text != "" {
		return f.Text
	}
	return f.Condition.String()
}

// TokenReader implements the xmlstream.Marshaler interface.
func (f Error) TokenReader() xml.TokenReader {
	inner := []xml.TokenReader{
		f.Condition.TokenReader(),
	}
	if f.Text != "" {
		var attrs []xml.Attr
		if f.Lang != "" {
			attrs = append(attrs, xml.Attr{
				Name:  xml.Name{Space: ns.XML, Local: "lang"},
				Value: f.Lang,
			})
		}
		inner = append(inner, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(f.Text)),
			xml.StartElement{
				Name: xml.Name{Space: "", Local: "text"},
				Attr: attrs,
			},
		))
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(inner...),
		xml.StartElement{
			Name: xml.Name{Space: ns.SASL, Local: "failure"},
		},
	)
}

// WriteXML implements the xmlstream.WriterTo interface.
func (f Error) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface for a Failure.
func (f Error) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := f.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// UnmarshalXML satisfies the xml.Unmarshaler interface for a Failure.
//
// If multiple text elements are present in the XML, UnmarshalXML selects the
// text element with an xml:lang attribute that exactly matches the language
// tag.
// If no language tag in the XML matches the behavior is undefined and may
// change at a later date or depending on the server order of tags.
func (f *Error) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		Condition Condition `xml:",any"`
		Text      []struct {
			Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
			Data string `xml:",chardata"`
		} `xml:"text"`
	}{}
	if err := d.DecodeElement(&decoded, &start); err != nil {
		return err
	}
	f.Condition = decoded.Condition
	for _, text := range decoded.Text {
		f.Lang = text.Lang
		f.Text = text.Data
		if text.Lang == f.Lang {
			break
		}
	}
	return nil
}
