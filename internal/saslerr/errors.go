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

// TokenReader implements the xmlstream.Marshaler interface.
func (c Condition) TokenReader() xml.TokenReader {
	if c == None || c >= Condition(len(_Condition_index)-1) {
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
	inner := []xml.TokenReader{
		f.Condition.TokenReader(),
	}
	if f.Text != "" {
		var attrs []xml.Attr
		if f.Lang != language.Und {
			attrs = append(attrs, xml.Attr{
				Name:  xml.Name{Space: ns.XML, Local: "lang"},
				Value: f.Lang.String(),
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
	tags := make([]language.Tag, 0, len(decoded.Text))
	data := make(map[language.Tag]string)
	for _, text := range decoded.Text {
		// Parse the language tag, skipping any that cannot be parsed.
		/* #nosec */
		tag, _ := language.Parse(text.Lang)
		tags = append(tags, tag)
		data[tag] = text.Data
	}
	tag, _, _ := language.NewMatcher(tags).Match(f.Lang)
	f.Lang = tag
	f.Text = data[tag]
	return nil
}
