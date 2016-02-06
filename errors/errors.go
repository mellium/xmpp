// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package errors

import (
	"encoding/xml"
)

// New returns an error that formats as the given text and marshals with the
// given XML name and with the text as chardata.
func New(name xml.Name, text string) error {
	return &ErrorXML{
		XMLName:  name,
		CharData: text,
	}
}

// ErrorXML is a trivial implementation of error intended to be marshalable and
// unmarshalable as XML.
type ErrorXML struct {
	XMLName  xml.Name
	InnerXML string `xml:",innerxml"`
	CharData string `xml:",chardata"`
}

// Satisfies the error interface and returns the error string.
func (e *ErrorXML) Error() string {
	return e.CharData
}
