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
	// If the payload to marshal is already a TokenReader, just return it.
	if r, ok := v.(xml.TokenReader); ok {
		return r, nil
	}

	return tokenDecoder(v)
}

func tokenDecoder(v interface{}) (*xml.Decoder, error) {
	var b bytes.Buffer
	err := xml.NewEncoder(&b).Encode(v)
	if err != nil {
		return nil, err
	}
	return xml.NewDecoder(&b), nil
}

// rawTokenReader maps a decoders RawToken method onto its Token method.
type rawTokenReader struct {
	*xml.Decoder
}

func (r rawTokenReader) Token() (xml.Token, error) {
	return r.RawToken()
}

// EncodeXML writes the XML encoding of v to the stream.
//
// See the documentation for xml.Marshal for details about the conversion of Go
// values to XML.
//
// If the stream is an xmlstream.Flusher, EncodeXML calls Flush before
// returning.
func EncodeXML(w xmlstream.TokenWriter, v interface{}) error {
	d, err := tokenDecoder(v)
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(w, rawTokenReader{Decoder: d})
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
	d, err := tokenDecoder(v)
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(w, rawTokenReader{Decoder: d})
	if err != nil {
		return err
	}

	if wf, ok := w.(xmlstream.Flusher); ok {
		return wf.Flush()
	}
	return nil
}
