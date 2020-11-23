// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package marshal contains functions for encoding structs as an XML token
// stream.
package marshal // import "mellium.im/xmpp/internal/marshal"

import (
	"bytes"
	"encoding/xml"

	"mellium.im/xmlstream"
)

// BUG(ssw): This package is very inefficient, see https://mellium.im/issue/38.

// TokenReader returns a reader for the XML encoding of v.
func TokenReader(v interface{}) (xml.TokenReader, error) {
	if r, ok := v.(xml.TokenReader); ok {
		return r, nil
	}

	var b bytes.Buffer
	err := xml.NewEncoder(&b).Encode(v)
	if err != nil {
		return nil, err
	}
	return xml.NewDecoder(&b), nil
}

// EncodeXML writes the XML encoding of v to the stream.
//
// See the documentation for xml.Marshal for details about the conversion of Go
// values to XML.
//
// If the stream is an xmlstream.Flusher, EncodeXML calls Flush before
// returning.
func EncodeXML(w xmlstream.TokenWriter, v interface{}) error {
	r, err := TokenReader(v)
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(w, r)
	if err != nil {
		return err
	}

	if wf, ok := w.(xmlstream.Flusher); ok {
		return wf.Flush()
	}
	return nil
}

// EncodeXMLElement writes the XML encoding of v to the stream, using start as
// the outermost tag in the encoding.
//
// See the documentation for xml.Marshal for details about the conversion of Go
// values to XML.
//
// If the stream is an xmlstream.Flusher, EncodeXMLElement calls Flush before
// returning.
func EncodeXMLElement(w xmlstream.TokenWriter, v interface{}, start xml.StartElement) error {
	r, err := TokenReader(v)
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(w, xmlstream.Wrap(r, start))
	if err != nil {
		return err
	}

	if wf, ok := w.(xmlstream.Flusher); ok {
		return wf.Flush()
	}
	return nil
}
