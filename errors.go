// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

// New returns an error that formats as the given text and marshals with the
// given XML name and with the text as chardata.
// func Error(name xml.Name, text string) error {
// 	return &ErrorXML{
// 		XMLName:  name,
// 		CharData: text,
// 	}
// }

// Error is a trivial implementation of error intended to be marshalable and
// unmarshalable as XML.
type Error struct {
	XMLName  xml.Name
	InnerXML string `xml:",innerxml"`
}

// Satisfies the error interface and returns the error string.
func (e Error) Error() string {
	return e.InnerXML
}
