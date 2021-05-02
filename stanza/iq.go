// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	XMLName xml.Name `xml:"iq"`
	ID      string   `xml:"id,attr"`
	To      jid.JID  `xml:"to,attr,omitempty"`
	From    jid.JID  `xml:"from,attr,omitempty"`
	Lang    string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    IQType   `xml:"type,attr"`
}

// UnmarshalIQError converts the provided XML token into an IQ.
// If the type of the IQ is "error" it unmarshals the entire payload and returns
// the error along with the original IQ.
func UnmarshalIQError(r xml.TokenReader, start xml.StartElement) (IQ, error) {
	iqStart, err := NewIQ(start)
	if err != nil {
		return iqStart, err
	}
	if iqStart.Type != ErrorIQ {
		return iqStart, nil
	}

	d := xml.NewTokenDecoder(r)
	var stanzaErr Error
	decodeErr := d.Decode(&stanzaErr)
	if decodeErr != nil {
		return iqStart, decodeErr
	}
	return iqStart, stanzaErr
}

// NewIQ unmarshals an XML token into a IQ.
func NewIQ(start xml.StartElement) (IQ, error) {
	v := IQ{
		XMLName: start.Name,
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "lang" && attr.Name.Space == ns.XML {
			v.Lang = attr.Value
			continue
		}
		if attr.Name.Space != "" && attr.Name.Space != start.Name.Space {
			continue
		}

		var err error
		switch attr.Name.Local {
		case "id":
			v.ID = attr.Value
		case "to":
			if attr.Value != "" {
				v.To, err = jid.Parse(attr.Value)
				if err != nil {
					return v, err
				}
			}
		case "from":
			if attr.Value != "" {
				v.From, err = jid.Parse(attr.Value)
				if err != nil {
					return v, err
				}
			}
		case "type":
			v.Type = IQType(attr.Value)
		}
	}
	return v, nil
}

// StartElement converts the IQ into an XML token.
func (iq IQ) StartElement() xml.StartElement {
	// Keep whatever namespace we're already using but make sure the localname is
	// "iq".
	name := iq.XMLName
	name.Local = "iq"

	attr := make([]xml.Attr, 0, 5)
	attr = append(attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: string(iq.Type)})
	if !iq.To.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: iq.To.String()})
	}
	if !iq.From.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "from"}, Value: iq.From.String()})
	}
	if iq.ID != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: iq.ID})
	}
	if iq.Lang != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: iq.Lang})
	}

	return xml.StartElement{
		Name: name,
		Attr: attr,
	}
}

// Wrap wraps the payload in a stanza.
//
// The resulting IQ may not contain an id or from attribute and thus may not be
// valid without further processing.
func (iq IQ) Wrap(payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, iq.StartElement())
}

// Result returns a token reader that wraps the first element from payload in an
// IQ stanza with the to and from attributes switched and the type set to
// ResultIQ.
func (iq IQ) Result(payload xml.TokenReader) xml.TokenReader {
	iq.Type = ResultIQ
	iq.From, iq.To = iq.To, iq.From
	return iq.Wrap(payload)
}

// Error returns a token reader that wraps the first element from payload in an
// IQ stanza with the to and from attributes switched and the type set to
// ErrorIQ.
func (iq IQ) Error(err Error) xml.TokenReader {
	iq.Type = ErrorIQ
	iq.From, iq.To = iq.To, iq.From
	return iq.Wrap(err.TokenReader())
}

// IQType is the type of an IQ stanza.
// It should normally be one of the constants defined in this package.
type IQType string

const (
	// GetIQ is used to query another entity for information.
	GetIQ IQType = "get"

	// SetIQ is used to provide data to another entity, set new values, and
	// replace existing values.
	SetIQ IQType = "set"

	// ResultIQ is sent in response to a successful get or set IQ.
	ResultIQ IQType = "result"

	// ErrorIQ is sent to report that an error occurred during the delivery or
	// processing of a get or set IQ.
	ErrorIQ IQType = "error"
)

// MarshalText ensures that the zero value for IQType is marshaled to XML as a
// valid IQ get request.
// It satisfies the encoding.TextMarshaler interface for IQType.
func (t IQType) MarshalText() ([]byte, error) {
	if t == "" {
		t = GetIQ
	}
	return []byte(t), nil
}
