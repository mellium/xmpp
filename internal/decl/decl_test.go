// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package decl_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/decl"
)

var skipTests = [...]struct {
	in  string
	out string
}{
	0: {},
	1: {in: "<a/>", out: "<a></a>"},
	2: {in: xml.Header + "<a/>", out: "\n<a></a>"},
	3: {in: `<?xml?><a/>`, out: "<a></a>"},
	4: {in: `<?sgml?><a/>`, out: "<?sgml?><a></a>"},
	5: {in: `<?xml?>`},
}

func TestDecl(t *testing.T) {
	for i, tc := range skipTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			d := decl.Skip(xml.NewDecoder(strings.NewReader(tc.in)))
			buf := &bytes.Buffer{}
			e := xml.NewEncoder(buf)
			if _, err := xmlstream.Copy(e, d); err != nil {
				t.Fatalf("Error copying tokens: %q", err)
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("Error flushing tokens: %q", err)
			}
			if s := buf.String(); s != tc.out {
				t.Errorf("Output does not match: want=%q, got=%q", tc.out, s)
			}
		})
	}
}

func TestImmediateEOF(t *testing.T) {
	d := decl.Skip(xmlstream.Token(xml.ProcInst{Target: "xml"}))

	for i := 0; i < 2; i++ {
		tok, err := d.Token()
		if err != io.EOF {
			t.Errorf("Expected EOF on %d but got %q", i, err)
		}
		if tok != nil {
			t.Errorf("Did not expect token on %d but got %T %[2]v", i, tok)
		}
	}
}
