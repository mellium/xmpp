// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmpp/styling"
)

var decoderTests = [...]struct {
	text   string
	bufs   []int
	reads  []string
	styles []styling.Style
	err    error
}{
	0: {err: io.EOF},
	1: {
		text:   "```ignored\next",
		bufs:   []int{len("```ignored\next")},
		reads:  []string{"```ignored\next"},
		styles: []styling.Style{styling.PreBlock},
		err:    io.EOF,
	},
	2: {
		text:   "```ignored\next",
		bufs:   []int{2, len("`ignored\next")},
		reads:  []string{"``", "`ignored\next"},
		styles: []styling.Style{styling.PreBlock, styling.PreBlock},
		err:    io.EOF,
	},
	3: {
		text:   "```",
		bufs:   []int{3},
		reads:  []string{"```"},
		styles: []styling.Style{styling.PreBlock},
		err:    io.EOF,
	},
	4: {
		text:   "line\n```",
		bufs:   []int{5, 3},
		reads:  []string{"line\n", "```"},
		styles: []styling.Style{0, styling.PreBlock},
		err:    io.EOF,
	},
	5: {
		text:   "line\n````",
		bufs:   []int{6, 4},
		reads:  []string{"line\n", "````"},
		styles: []styling.Style{0, styling.PreBlock},
		err:    io.EOF,
	},
	6: {
		text:   "line\n````\ntest\n```",
		bufs:   []int{6, len("````\ntest\n```")},
		reads:  []string{"line\n", "````\ntest\n```"},
		styles: []styling.Style{0, styling.PreBlock},
		err:    io.EOF,
	},
	7: {
		text:   "line\n````\ntest\n```\ntest",
		bufs:   []int{6, len("````\ntest\n```"), 5},
		reads:  []string{"line\n", "````\ntest\n```", "\ntest"},
		styles: []styling.Style{0, styling.PreBlock, 0},
		err:    io.EOF,
	},
	8: {
		text:   "line\n````\nte```st\n```\ntest",
		bufs:   []int{6, len("````\nte```st\n```"), 5},
		reads:  []string{"line\n", "````\nte```st\n```", "\ntest"},
		styles: []styling.Style{0, styling.PreBlock, 0},
		err:    io.EOF,
	},
}

func TestDecoder(t *testing.T) {
	for i, tc := range decoderTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			parser := styling.NewParser(strings.NewReader(tc.text))

			var err error
			var n int
			var i int
			var out []byte
			for ; i < len(tc.bufs); i++ {
				curStyle := tc.styles[i]
				b := make([]byte, tc.bufs[i])
				n, err = parser.Read(b)
				if tc.reads[i] != string(b[:n]) {
					t.Errorf("Bad read: want=%q, got=%q", tc.reads[i], b[:n])
				}
				if s := parser.Style(); s != curStyle {
					t.Errorf("Wrong style: want=%q, got=%q", curStyle, s)
				}
				if err != nil {
					break
				}
				out = append(out, b[:n]...)
			}

			if err == nil {
				n, err = parser.Read(make([]byte, 10))
				if n != 0 {
					t.Errorf("Read after final returned unexpected bytes")
				}
			}

			if string(out) != tc.text {
				t.Errorf("Unexpected text: want=%q, got=%q", tc.text, out)
			}
			if i != len(tc.bufs) {
				t.Errorf("Wrong number of reads: want=%d, got=%d", len(tc.bufs), i)
			}
			if err != tc.err {
				t.Errorf("Unexpected error: want=%q, got=%q", tc.err, err)
			}
		})
	}
}
