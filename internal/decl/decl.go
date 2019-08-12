// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package decl contains functionality related to XML declarations.
package decl

import (
	"encoding/xml"
)

const (
	// XMLHeader is an XML header like the one in encoding/xml but without a
	// newline at the end.
	XMLHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type skipper struct {
	r       xml.TokenReader
	started bool
}

// Token implements xml.TokenReader for Reader.
func (r *skipper) Token() (xml.Token, error) {
	tok, err := r.r.Token()
	if tok != nil && !r.started {
		r.started = true
		if proc, ok := tok.(xml.ProcInst); ok && proc.Target == "xml" {
			if err != nil {
				return nil, err
			}
			return r.r.Token()
		}
	}
	return tok, err
}

// Skip wraps a token reader and skips any XML declaration.
func Skip(r xml.TokenReader) xml.TokenReader {
	return &skipper{r: r}
}
