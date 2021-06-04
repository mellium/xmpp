// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build gofuzzbeta

package styling_test

import (
	"bytes"
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
		for {
			tok, err := d.Token()
			if err == io.EOF {
				if n+len(tok.Data) != len(doc) {
					t.Fatalf("got early EOF at %d for input:\n%v", n, doc)
				}
				break
			}
			if err != nil {
				t.Fatalf("error decoding: %v\nOriginal bytes:\n%v", err, doc)
				break
			}
			if !bytes.Equal([]byte(doc[n:n+len(tok.Data)]), tok.Data) {
				t.Fatalf("output bytes did not equal input bytes at %d for input:\n%v", n, doc)
			}
			n += len(tok.Data)
		}
	})
}
