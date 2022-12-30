// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package saslerr provides error conditions for the XMPP profile of SASL as
// defined by RFC 6120 §6.5.
package saslerr // import "mellium.im/xmpp/internal/saslerr"

//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -type=Condition -linecomment

import (
	"encoding/xml"

	"golang.org/x/text/language"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
)

// Condition represents a SASL error condition that can be encapsulated by a
// <failure/> element.
type Condition uint16

// Standard SASL error conditions.
const (
	// None is a special condition that is present only if a defined condition was
	// not present. Its use violates the spec.
	None                 Condition = iota // none
	Aborted                               // aborted
	AccountDisabled                       // account-disabled
	CredentialsExpired                    // credentials-expired
	EncryptionRequired                    // encryption-required
	IncorrectEncoding                     // incorrect-encoding
	InvalidAuthzID                        // invalid-authzid
	InvalidMechanism                      // invalid-mechanism
	MalformedRequest                      // malformed-request
	MechanismTooWeak                      // mechanism-too-weak
	NotAuthorized                         // not-authorized
	TemporaryAuthFailure                  // temporary-auth-failure
)

// Failure represents a SASL error that is marshalable to XML.
type Failure struct {
	Condition Condition
	Lang      language.Tag
	Text      string
}

// Error satisfies the error interface for a Failure. It returns the text string
// if set, or the condition otherwise.
func (f Failure) Error() string {
	if f.Text != "" {
		return f.Text
	}
	return f.Condition.String()
}

// TokenReader implements the xmlstream.Marshaler interface.
func (f Failure) TokenReader() xml.TokenReader {
	var cond xml.TokenReader
	if f.Condition != None {
		cond = xmlstream.Wrap(
			nil,
			xml.StartElement{
				Name: xml.Name{Local: f.Condition.String()},
			},
		)
	}
	inner := []xml.TokenReader{
		cond,
	}
	if f.Text != "" {
		inner = append(inner, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(f.Text)),
			xml.StartElement{
				Name: xml.Name{Space: "", Local: "text"},
				Attr: []xml.Attr{
					{
						Name:  xml.Name{Space: ns.XML, Local: "lang"},
						Value: f.Lang.String(),
					},
				},
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
func (f Failure) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface for a Failure.
func (f Failure) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := f.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// UnmarshalXML satisfies the xml.Unmarshaler interface for a Failure. If
// multiple text elements are present in the XML and the Failure struct already
// has a language tag set, UnmarshalXML selects the text element with an
// xml:lang attribute that most closely matches the features language tag. If no
// language tag is present, UnmarshalXML selects a text element with an xml:lang
// attribute of "und" if present, behavior is undefined otherwise (it will pick
// the tag that most closely matches "und", whatever that means).
func (f *Failure) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		Condition struct {
			XMLName xml.Name
		} `xml:",any"`
		Text []struct {
			Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
			Data string `xml:",chardata"`
		} `xml:"text"`
	}{}
	if err := d.DecodeElement(&decoded, &start); err != nil {
		return err
	}
	switch decoded.Condition.XMLName.Local {
	case "not-authorized":
		f.Condition = NotAuthorized
	case "aborted":
		f.Condition = Aborted
	case "account-disabled":
		f.Condition = AccountDisabled
	case "credentials-expired":
		f.Condition = CredentialsExpired
	case "encryption-required":
		f.Condition = EncryptionRequired
	case "incorrect-encoding":
		f.Condition = IncorrectEncoding
	case "invalid-authzid":
		f.Condition = InvalidAuthzID
	case "invalid-mechanism":
		f.Condition = InvalidMechanism
	case "malformed-request":
		f.Condition = MalformedRequest
	case "mechanism-too-weak":
		f.Condition = MechanismTooWeak
	case "temporary-auth-failure":
		f.Condition = TemporaryAuthFailure
	default:
		f.Condition = None
	}
	tags := make([]language.Tag, 0, len(decoded.Text))
	data := make(map[language.Tag]string)
	for _, text := range decoded.Text {
		// Parse the language tag, skipping any that cannot be parsed.
		tag, err := language.Parse(text.Lang)
		if err != nil {
			continue
		}
		tags = append(tags, tag)
		data[tag] = text.Data
	}
	tag, _, _ := language.NewMatcher(tags).Match(f.Lang)
	f.Lang = tag
	f.Text = data[tag]
	return nil
}
