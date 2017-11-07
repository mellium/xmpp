// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"strconv"
	"testing"

	"mellium.im/xmpp/styling"
)

var transformTests = [...]struct {
	input       string
	html        string
	htmlNoStyle string
	markdown    string
}{
	0: {},
	1: {
		input:       "```\n```",
		html:        "<pre>```\n```</pre>",
		htmlNoStyle: "<pre></pre>",
		markdown:    "```\n```",
	},
	2: {
		input:       "```\n```\n",
		html:        "<pre>```\n```</pre>\n",
		htmlNoStyle: "<pre></pre>\n",
		markdown:    "```\n```\n",
	},
	3: {
		input: "```" + `
		This is *preformatted* text!
	test.
` + "```hooray",
		html: "<pre>```" + `
		This is *preformatted* text!
	test.
` + "```</pre>hooray",
		htmlNoStyle: `<pre>		This is *preformatted* text!
	test.
</pre>hooray`,
		markdown: "```" + `
		This is *preformatted* text!
	test.
` + "```hooray",
	},
}

func TestTransform(t *testing.T) {
	for i, tc := range transformTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("HTML", func(t *testing.T) {
				transformer := styling.HTML()
				out := transformer.String(tc.input)
				if out != tc.html {
					t.Errorf("Bad output:\nwant=\"%s\"\ngot=\"%s\"\n", tc.html, out)
				}
			})
			t.Run("HTMLNoStyle", func(t *testing.T) {
				transformer := styling.HTMLNoStyle()
				out := transformer.String(tc.input)
				if out != tc.htmlNoStyle {
					t.Errorf("Bad output:\nwant=\"%s\"\ngot=\"%s\"\n", tc.htmlNoStyle, out)
				}
			})
			t.Run("Markdown", func(t *testing.T) {
				transformer := styling.Markdown()
				out := transformer.String(tc.input)
				if out != tc.markdown {
					t.Errorf("Bad output:\nwant=\"%s\"\ngot=\"%s\"\n", tc.markdown, out)
				}
			})
		})
	}
}
