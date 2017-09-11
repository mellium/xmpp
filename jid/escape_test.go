// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid

import (
	"fmt"
	"testing"

	"golang.org/x/text/transform"
)

var _ transform.SpanningTransformer = (*escapeMapping)(nil)

const allescaped = `\20\22\26\27\2f\3a\3c\3e\40\5c`

var escapeTestCases = [...]struct {
	unescaped, escaped string
	atEOF              bool
	span               int
	err, spanErr       error
}{
	0: {escape, allescaped, true, 0, nil, transform.ErrEndOfSpan},
	1: {escape, allescaped, false, 0, nil, transform.ErrEndOfSpan},
	2: {`nothingtodohere`, `nothingtodohere`, true, 15, nil, nil},
	3: {`nothingtodohere`, `nothingtodohere`, false, 15, nil, nil},
	4: {"", "", true, 0, nil, nil},
	5: {"", "", false, 0, nil, nil},
	6: {`a `, `a\20`, true, 1, nil, transform.ErrEndOfSpan},
}

var unescapeTestCases = [...]struct {
	escaped, unescaped string
	atEOF              bool
	span               int
	err, spanErr       error
}{
	0: {allescaped, escape, true, 0, nil, transform.ErrEndOfSpan},
	1: {`a\20`, `a `, true, 1, nil, transform.ErrEndOfSpan},
	2: {`a\`, `a\`, true, 2, nil, nil},
	3: {`a\`, `a`, false, 1, transform.ErrShortSrc, transform.ErrShortSrc},
	4: {`nothingtodohere`, `nothingtodohere`, true, 15, nil, nil},
	5: {`nothingtodohere`, `nothingtodohere`, false, 15, nil, nil},
	6: {`a\a\20`, `a\a `, false, 3, nil, transform.ErrEndOfSpan},
	7: {`aa\2`, `aa\2`, true, 4, nil, nil},
	8: {`aa\2`, `aa`, false, 2, transform.ErrShortSrc, transform.ErrShortSrc},
}

func TestUnescape(t *testing.T) {
	for i, tc := range unescapeTestCases {
		t.Run(fmt.Sprintf("Transform/%d", i), func(t *testing.T) {
			buf := make([]byte, 100)
			switch nDst, _, err := Unescape.Transform(buf, []byte(tc.escaped), tc.atEOF); {
			case err != tc.err:
				t.Errorf("Unexpected error, got=%v, want=%v", err, tc.err)
			case string(buf[:nDst]) != tc.unescaped:
				t.Errorf("Unescaped localpart should be `%s` but got: `%s`", tc.unescaped, string(buf[:nDst]))
			}
		})
		t.Run(fmt.Sprintf("String/%d", i), func(t *testing.T) {
			if tc.err != nil {
				t.Skip("Skipping test with expected error")
			}
			if unescaped := Unescape.String(tc.escaped); unescaped != tc.unescaped {
				t.Errorf("Unescaped localpart should be `%s` but got: `%s`", tc.unescaped, unescaped)
			}
		})
		t.Run(fmt.Sprintf("Bytes/%d", i), func(t *testing.T) {
			if tc.err != nil {
				t.Skip("Skipping test with expected error")
			}
			if unescaped := Unescape.Bytes([]byte(tc.escaped)); string(unescaped) != tc.unescaped {
				t.Errorf("Unescaped localpart should be `%s` but got: `%s`", tc.unescaped, unescaped)
			}
		})
		t.Run(fmt.Sprintf("Span/%d", i), func(t *testing.T) {
			switch n, err := Unescape.Span([]byte(tc.escaped), tc.atEOF); {
			case err != tc.spanErr:
				t.Errorf("Unexpected error, got=%v, want=%v", err, tc.spanErr)
			case n != tc.span:
				t.Errorf("Unexpected span, got=%d, want=%d", n, tc.span)
			}
		})
	}
}

func TestEscape(t *testing.T) {
	for i, tc := range escapeTestCases {
		t.Run(fmt.Sprintf("Transform/%d", i), func(t *testing.T) {
			switch e, _, err := transform.String(Escape, tc.unescaped); {
			case err != tc.err:
				t.Errorf("Unexpected error, got=%v, want=%v", err, tc.err)
			case e != tc.escaped:
				t.Errorf("Escaped localpart should be `%s` but got: `%s`", tc.escaped, e)
			}
		})
		t.Run(fmt.Sprintf("Span/%d", i), func(t *testing.T) {
			switch n, err := Escape.Span([]byte(tc.unescaped), tc.atEOF); {
			case err != tc.spanErr:
				t.Errorf("Unexpected error, got=%v, want=%v", err, tc.spanErr)
			case n != tc.span:
				t.Errorf("Unexpected span, got=%d, want=%d", n, tc.span)
			}
		})
	}
}

// TODO: Malloc tests may be flakey under GCC until it improves its escape
//       analysis.

func TestEscapeMallocs(t *testing.T) {
	src := []byte(escape)
	dst := make([]byte, len(src)+18)

	if n := testing.AllocsPerRun(1000, func() { Escape.Transform(dst, src, true) }); n > 0 {
		t.Errorf("got %f allocs, want 0", n)
	}
}

func TestUnescapeMallocs(t *testing.T) {
	src := []byte(allescaped)
	dst := make([]byte, len(src)/3)

	if n := testing.AllocsPerRun(1000, func() { Unescape.Transform(dst, src, true) }); n > 0 {
		t.Errorf("got %f allocs, want 0", n)
	}
}
