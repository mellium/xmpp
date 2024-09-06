// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package decl_test

import (
	"encoding/xml"
	"io"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/decl"
	"mellium.im/xmpp/internal/xmpptest"
)

var skipTests = []xmpptest.TransformerTestCase{
	0: {},
	1: {In: "<a/>", Out: "<a></a>"},
	2: {In: xml.Header + "<a/>", Out: "\n<a></a>"},
	3: {In: `<?xml?><a/>`, Out: "<a></a>"},
	4: {In: `<?sgml?><a/>`, Out: "<?sgml?><a></a>"},
	5: {In: `<?xml?>`},
}

func TestDecl(t *testing.T) {
	xmpptest.RunTransformerTests(t, decl.Skip, skipTests)
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

var trimSpaceTests = []xmpptest.TransformerTestCase{
	0: {},
	1: {In: "\t\n\r <a/>", Out: "<a></a>"},
	2: {In: "a b<a/>", Out: "a b<a></a>"},
	3: {In: "\n<?foo bar ?><a/>\n", Out: "<?foo bar ?><a></a>\n"},
	4: {In: " \n a b<a/>", Out: " \n a b<a></a>"},
	5: {In: " \n ", Out: ""},
	6: {InStream: xmlstream.ReaderFunc(func() (xml.Token, error) {
		// Concurrent EOF should also be trimmed.
		return xml.CharData(" \n"), io.EOF
	}), Out: ""},
}

func TestTrimSpace(t *testing.T) {
	xmpptest.RunTransformerTests(t, decl.TrimLeftSpace, trimSpaceTests)
}
