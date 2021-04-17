// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/xmpp/stream"
)

// Errors related to stream handling
var (
	ErrUnknownStreamElement = errors.New("xmpp: unknown stream level element")
	ErrUnexpectedRestart    = errors.New("xmpp: unexpected stream restart")
)

type reader struct {
	r xml.TokenReader
}

func (r reader) Token() (xml.Token, error) {
	tok, err := r.r.Token()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case xml.StartElement:
		if t.Name.Space != stream.NS {
			return tok, err
		}

		// Handle stream errors and unknown stream namespaced tokens first, before
		// delegating to the normal handler.
		switch t.Name.Local {
		case "error":
			e := stream.Error{}
			err = xml.NewTokenDecoder(r.r).DecodeElement(&e, &t)
			if err != nil {
				return nil, err
			}
			return nil, e
		case "stream":
			// Special case returning a nice error here.
			return nil, ErrUnexpectedRestart
		default:
			return nil, ErrUnknownStreamElement
		}
	case xml.EndElement:
		if t.Name.Space != stream.NS {
			return tok, err
		}

		// If this is a stream end element, we're done.
		if t.Name.Local == "stream" {
			return nil, io.EOF
		}

		// If this is a stream level end element but not </stream:stream>,
		// something is really weird…
		return nil, stream.BadFormat
	case xml.CharData:
		// Pass chardata through. We ensure that any chardata at the top level of
		// the stream is only whitespace elsewhere.
		return tok, err
	}
	// Other XML tokens are forbidden.
	return tok, fmt.Errorf("invalid token type: %T", tok)
}

// Reader returns a token reader that handles stream level tokens on an already
// established stream.
func Reader(r xml.TokenReader) xml.TokenReader {
	return reader{r: r}
}
