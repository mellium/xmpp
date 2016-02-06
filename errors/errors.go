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
	return &errorXML{
		XMLName:  name,
		CharData: text,
	}
}

// errorXML is a trivial implementation of error intended to be marshalable and
// unmarshalable as XML.
type errorXML struct {
	XMLName  xml.Name
	InnerXML string `xml:",innerxml"`
	CharData string `xml:",chardata"`
}

// Satisfies the error interface and returns the error string.
func (e *errorXML) Error() string {
	return e.CharData
}
