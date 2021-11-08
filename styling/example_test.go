// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"fmt"
	"html"
	"io"
	"strings"

	"mellium.im/xmpp/styling"
)

func Example_html() {
	r := strings.NewReader(`The full title is
_Twelfth Night, or What You Will_
but *most* people shorten it.`)
	d := styling.NewDecoder(r)

	var out strings.Builder
	out.WriteString("<!doctype HTML>\n")
	for d.Next() {
		tok := d.Token()
		mask := d.Style()

		switch {
		case mask&styling.SpanEmphStart == styling.SpanEmphStart:
			out.WriteString("<em><code>")
			out.Write(tok.Data)
			out.WriteString("</code>")
		case mask&styling.SpanStrongStart == styling.SpanStrongStart:
			out.WriteString("<strong><code>")
			out.Write(tok.Data)
			out.WriteString("</code>")
		case mask&styling.SpanEmphEnd == styling.SpanEmphEnd:
			out.WriteString("<code>")
			out.Write(tok.Data)
			out.WriteString("</code></em>")
		case mask&styling.SpanStrongEnd == styling.SpanStrongEnd:
			out.WriteString("<code>")
			out.Write(tok.Data)
			out.WriteString("</code></strong>")
			// TODO: no other styles implemented to keep the example short.
		default:
			out.WriteString(html.EscapeString(string(tok.Data)))
		}
	}

	err := d.Err()
	if err != nil && err != io.EOF {
		out.WriteString("<mark>")
		out.WriteString(html.EscapeString(fmt.Sprintf("Error encountered while parsing tokens: %v", err)))
		out.WriteString("</mark>")
	}
	fmt.Println(out.String())

	// Output:
	// <!doctype HTML>
	// The full title is
	// <em><code>_</code>Twelfth Night, or What You Will<code>_</code></em>
	// but <strong><code>*</code>most<code>*</code></strong> people shorten it.
}
