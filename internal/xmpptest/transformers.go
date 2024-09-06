// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest

import (
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
)

// TransformerTestCase is a data driven test for XML transformers.
type TransformerTestCase struct {
	In  string
	Out string

	// If InStream is not nil, it will be used instead of "In" and should result
	// in tokens matching Out.
	InStream xml.TokenReader
}

func RunTransformerTests(t *testing.T, T xmlstream.Transformer, tcs []TransformerTestCase) {
	for i, tc := range tcs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var d xml.TokenReader
			if tc.InStream != nil {
				d = T(tc.InStream)
			} else {
				d = T(xml.NewDecoder(strings.NewReader(tc.In)))
			}
			buf := &strings.Builder{}
			e := xml.NewEncoder(buf)
			if _, err := xmlstream.Copy(e, d); err != nil {
				t.Fatalf("error copying tokens: %q", err)
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("error flushing tokens: %q", err)
			}
			if s := buf.String(); s != tc.Out {
				t.Errorf("output does not match: want=%q, got=%q", tc.Out, s)
			}
		})
	}
}
