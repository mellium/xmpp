// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"errors"
	"strings"
)

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	Stanza

	Type    iqType   `xml:"type,attr"`
	XMLName xml.Name `xml:"iq"`
}

type iqType int

const (
	// A Get IQ is used to query another entity for information.
	Get iqType = iota

	// A Set IQ is used to provide data to another entity, set new values, and
	// replace existing values.
	Set

	// A Result IQ is sent in response to a successful get or set IQ.
	Result

	// An Error IQ is sent to report that an error occured during the delivery or
	// processing of a get or set IQ.
	Error
)

func (t iqType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.ToLower(t.String())}, nil
}

func (t *iqType) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "get":
		*t = Get
	case "set":
		*t = Set
	case "result":
		*t = Result
	case "error":
		*t = Error
	default:
		// TODO: This should be a stanza error with the bad-request condition.
		return errors.New("bad-request")
	}
	return nil
}
