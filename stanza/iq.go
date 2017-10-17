// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// WrapIQ wraps a payload in an IQ stanza.
// The resulting IQ does not contain an id or from attribute and is thus not
// valid without further processing.
func WrapIQ(to *jid.JID, typ IQType, payload xml.TokenReader) xml.TokenReader {
	attr := []xml.Attr{{Name: xml.Name{Local: "type"}, Value: string(typ)}}
	if to != nil {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: to.String()})
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
	To      *jid.JID `xml:"to,attr,omitempty"`
	From    *jid.JID `xml:"from,attr,omitempty"`
	Lang    string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    IQType   `xml:"type,attr"`
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
