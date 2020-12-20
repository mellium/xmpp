// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build fuzz

// This tool is meant to excersize the styling code with random documents that
// are likely to contain a high concentration of styling characters.
// This will hopeful tease out any panics that are hidden throughout the code,
// or any places where the output of a token does not exactly match the input
// (due to an off-by-one error causing the output to be truncated, for example).
// No care has been taken to try and make this fast, so it is very slow and does
// not get run with the normal tests.

package styling_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	"mellium.im/xmpp/styling"
)

// The strings in this list will be selected with a higher probability than
// other random runes to ensure that we add lots of styling directives that will
// tease out any problems with odd combinations of them in the decoder.
var highProbabilityAlphabet = []string{"```", " ", ">", "`", "*", "_", "\n"}

const (
	// The maximum length of generated documents.
	documentLength = 1024

	// The number of documents to generate.
	iterations = 1 << 21

	// An ~1/3 chance of selecting something from the high probability alphabet
	// (which is mostly styling directives).
	probabilityOfDirective = 3
)

func randDoc(size int) []byte {
	var b bytes.Buffer
	for b.Len() < size {
		l := len(highProbabilityAlphabet)
		choice := rand.Intn(l * probabilityOfDirective)
		if choice >= len(highProbabilityAlphabet) {
			b.WriteRune(rune(rand.Uint32()))
			continue
		}
		b.WriteString(highProbabilityAlphabet[choice])
	}
	b.Truncate(size)
	return b.Bytes()
}

func TestFuzz(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < iterations; i++ {
		doc := randDoc(rand.Intn(documentLength))
		d := styling.NewDecoder(bytes.NewReader(doc))
		n := 0
		for {
			tok, err := d.Token()
			if err == io.EOF {
				if n+len(tok.Data) != len(doc) {
					t.Fatalf("Got early EOF at %d for input:\n%v", n, doc)
				}
				break
			}
			if err != nil {
				t.Fatalf("Error decoding: %v\nOriginal bytes:\n%v", err, doc)
				break
			}
			if !bytes.Equal(doc[n:n+len(tok.Data)], tok.Data) {
				t.Fatalf("Output bytes did not equal input bytes at %d for input:\n%v", n, doc)
			}
			n += len(tok.Data)
		}
	}
}
