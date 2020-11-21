// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest

import (
	"encoding/xml"
	"io"
)

// Tokens is a slice of XML tokens that can also act as an xml.TokenReader by
// popping tokens from itself.
// This is useful for testing contrived scenarios where the tokens cannot be
// constructed using an xml.Decoder because the stream to be tested violates the
// well-formedness rules of XML or otherwise would result in an error from the
// decoder.
type Tokens []xml.Token

func (r *Tokens) Token() (xml.Token, error) {
	if len(*r) == 0 {
		return nil, io.EOF
	}

	var t xml.Token
	t, *r = (*r)[0], (*r)[1:]
	return t, nil
}
