// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"mellium.im/xmpp/styling"
)

func FuzzParseDocument(f *testing.F) {
	f.Add([]byte{'*'})
	f.Add([]byte{'_'})
	f.Add([]byte{'`'})
	f.Add([]byte("```"))
	f.Add([]byte{'~'})
	f.Add([]byte{'>'})
	f.Add([]byte{'\n'})
	f.Fuzz(func(t *testing.T, doc []byte) {
		d := styling.NewDecoder(bytes.NewReader(doc))
		n := 0
		for d.Next() {
			tok := d.Token()
			if !bytes.Equal([]byte(doc[n:n+len(tok.Data)]), tok.Data) {
				t.Fatalf("output bytes did not equal input bytes at %d for input:\n%v", n, doc)
			}
			n += len(tok.Data)
		}
		switch err := d.Err(); {
		case errors.Is(err, io.EOF):
			if n != len(doc) {
				t.Fatalf("got early EOF at %d", n)
			}
		case err == nil:
		default:
			t.Fatalf("error decoding: %v\nOriginal bytes:\n%v", err, doc)
		}
	})
}
