// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"errors"

	"mellium.im/xmpp/jid"
)

// Errors returned by the stanza package.
var (
	ErrEmptyIQType = errors.New("stanza: empty IQ type")
)

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	XMLName xml.Name `xml:"iq"`
	ID      string   `xml:"id,attr"`
	To      *jid.JID `xml:"to,attr"`
	From    *jid.JID `xml:"from,attr"`
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

// MarshalXMLAttr satisfies the xml.MarshalerAttr interface for IQType.
// It returns ErrEmptyIQType when trying to marshal a IQ stanza with an empty
// type attribute.
func (t IQType) MarshalXMLAttr(name xml.Name) (attr xml.Attr, err error) {
	s := string(t)
	if s == "" {
		return attr, ErrEmptyIQType
	}
	attr.Name = name
	attr.Value = s
	return attr, nil
}
