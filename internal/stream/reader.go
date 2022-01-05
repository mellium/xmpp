// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmpp/stream"
)

// Errors related to stream handling
var (
	ErrUnknownStreamElement = errors.New("xmpp: unknown stream level element")
	ErrUnexpectedRestart    = errors.New("xmpp: unexpected stream restart")
)

type reader struct {
	r     xml.TokenReader
	depth uint64
}

func (r *reader) Token() (xml.Token, error) {
	tok, err := r.r.Token()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case xml.CharData:
		if r.depth == 0 {
			if len(bytes.TrimLeft(t, " \t\r\n")) != 0 {
				// Whitespace is allowed, but anything else at the top of the stream is
				// disallowed.
				return t, errors.New("xmpp: unexpected stream-level chardata")
			}
		}
	case xml.StartElement:
		r.depth++
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
		r.depth--
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
	case xml.ProcInst:
		return nil, errors.New("disallowed XML proc inst encountered")
	case xml.Comment:
		return nil, errors.New("disallowed XML comment encountered")
	case xml.Directive:
		return nil, errors.New("disallowed XML directive encountered")
	}
	return tok, err
}

// Reader returns a token reader that handles stream level tokens on an already
// established stream.
func Reader(r xml.TokenReader) xml.TokenReader {
	return &reader{r: r}
}
