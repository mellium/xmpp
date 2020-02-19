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

// WrapIQ wraps a payload in an IQ stanza.
// The resulting IQ may not contain an id or from attribute and thus may not be
// valid without further processing.
func WrapIQ(iq IQ, payload xml.TokenReader) xml.TokenReader {
	attr := []xml.Attr{
		{Name: xml.Name{Local: "type"}, Value: string(iq.Type)},
	}

	if !iq.To.Equal(jid.JID{}) {
		to, err := iq.To.MarshalXMLAttr(xml.Name{Space: "", Local: "to"})
		if err == nil && to.Value != "" {
			attr = append(attr, to)
		}
	}
	if !iq.From.Equal(jid.JID{}) {
		from, err := iq.From.MarshalXMLAttr(xml.Name{Space: "", Local: "from"})
		if err == nil && from.Value != "" {
			attr = append(attr, from)
		}
	}

	if iq.Lang != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "lang", Space: ns.XML}, Value: iq.Lang})
	}
	if iq.ID != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: iq.ID})
	}

	return xmlstream.Wrap(payload, xml.StartElement{
		Name: xml.Name{Local: "iq"},
		Attr: attr,
	})
}

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

// Result returns a token reader that wraps the first element from payload in an
// IQ stanza with the to and from attributes switched and the type set to
// ResultIQ.
func (iq IQ) Result(payload xml.TokenReader) xml.TokenReader {
	return WrapIQ(IQ{
		ID:   iq.ID,
		To:   iq.From,
		From: iq.To,
		Type: ResultIQ,
	}, payload)
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
