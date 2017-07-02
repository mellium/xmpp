// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package codec

import (
	"encoding/xml"
)

// A Codec is both a Decoder and an Encoder and can handle both sides of an XMPP
// stream.
type Codec interface {
	Decoder
	Encoder
}

// A Decoder is anything that can be used to decode an XML token stream
// (including an *xml.Decoder).
type Decoder interface {
	DecodeElement(v interface{}, start *xml.StartElement) error
	Decode(v interface{}) error
	Skip() error
	Token() (xml.Token, error)
}

// An Encoder is anything that can be used to encode an XML token stream
// (including an *xml.Encoder)
type Encoder interface {
	EncodeElement(v interface{}, start xml.StartElement) error
	EncodeToken(t xml.Token) error
	Encode(v interface{}) error
	Flush() error
}
