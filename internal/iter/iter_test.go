// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package iter_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/iter"
)

var (
	intStart = xml.StartElement{Name: xml.Name{Local: "int"}, Attr: []xml.Attr{}}
	fooStart = xml.StartElement{Name: xml.Name{Local: "foo"}, Attr: []xml.Attr{}}
)

var iterTests = [...]struct {
	in  string
	out [][]xml.Token
	err error
}{
	0: {in: `<a></a>`},
	1: {
		in: `<nums><int>1</int><foo/></nums>`,
		out: [][]xml.Token{
			{intStart, xml.CharData("1"), intStart.End()},
			{fooStart, fooStart.End()},
		},
	},
}

func TestIter(t *testing.T) {
	for i, tc := range iterTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			i := i
			_ = i
			d := xml.NewDecoder(strings.NewReader(tc.in))
			// Discard the opening tag.
			if _, err := d.Token(); err != nil {
				t.Fatalf("Error popping initial token: %q", err)
			}
			iter := iter.New(d)
			out := [][]xml.Token{}
			for iter.Next() {
				start, r := iter.Current()
				toks, err := xmlstream.ReadAll(r)
				if err != nil {
					t.Fatalf("Error reading tokens: %q", err)
				}
				if start != nil {
					toks = append([]xml.Token{start.Copy()}, toks...)
				}
				out = append(out, toks)
			}
			if err := iter.Err(); err != tc.err {
				t.Errorf("Wrong error: want=%q, got=%q", tc.err, err)
			}
			if err := iter.Close(); err != nil {
				t.Errorf("Error closing iter: %q", err)
			}

			// Check that the entire token stream was consumed and we didn't leave it
			// in a partially consumed state.
			if tok, err := d.Token(); err != io.EOF || tok != nil {
				t.Errorf("Expected token stream to be consumed, got token %+v, with err %q", tok, err)
			}

			// Don't try to compare nil and empty slice with DeepEqual
			if len(out) == 0 && len(tc.out) == 0 {
				return
			}

			if fmt.Sprintf("%#v", out) != fmt.Sprintf("%#v", tc.out) {
				t.Errorf("Wrong output:\nwant=\n%#v,\ngot=\n%#v", tc.out, out)
			}
		})
	}
}

type recordCloser struct {
	called bool
}

func (c *recordCloser) Close() error {
	c.called = true
	return nil
}

func TestIterClosesInner(t *testing.T) {
	recorder := &recordCloser{}
	rc := struct {
		xml.TokenReader
		io.Closer
	}{
		TokenReader: xml.NewDecoder(strings.NewReader(`<nums><int>1</int><foo/></nums>`)),
		Closer:      recorder,
	}
	iter := iter.New(rc)
	err := iter.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !recorder.called {
		t.Errorf("Expected iter to close the inner reader if it is a TokenReadCloser")
	}
}
