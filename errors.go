// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"mellium.im/xmpp/jid"
)

//go:generate stringer -type=errorType

type errorType int

const (
	// An error with type Auth indicates that an operation should be retried after
	// providing credentials.
	Auth errorType = iota

	// An error with type Cancel indicates that the error cannot be remedied and
	// the operation should not be retried.
	Cancel

	// An error with type Continue indicates that the operation can proceed (the
	// condition was only a warning).
	Continue

	// An error with type Modify indicates that the operation can be retried after
	// changing the data sent.
	Modify

	// An error with type Wait is temporary and may be retried after waiting.
	Wait
)

// StanzaError is an implementation of error intended to be marshalable and
// unmarshalable as XML.
type StanzaError struct {
	XMLName   xml.Name
	By        jid.JID `xml:"by,attr,omitempty"`
	Type      string  `xml:"type,attr"`
	Condition string  `xml:"-"`
	Text      struct {
		Lang     string `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
		CharData string `xml:",chardata"`
	} `xml:"text urn:ietf:params:xml:ns:xmpp-stanzas"`
	InnerXML string `xml:",innerxml"`
}

// Error satisfies the error interface and returns the condition.
func (e StanzaError) Error() string {
	return e.Condition
}
