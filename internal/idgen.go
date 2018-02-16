// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package internal

import (
	"crypto/rand"
	"fmt"
	"io"
)

// IDLen is the standard length of stanza identifiers in bytes.
const IDLen = 16

// TODO: This will be called a lot, and probably needs to be faster than we can
//       get when reading from getrandom(2). Should we use a fast userspace
//       CSPRNG and just seed with data from the OS?

// RandomID generates a new random identifier of length IDLen. If the OS's
// entropy pool isn't initialized, or we can't generate random numbers for some
// other reason, panic.
func RandomID() string {
	return randomID(IDLen, rand.Reader)
}

func randomID(n int, r io.Reader) string {
	b := make([]byte, (n/2)+(n&1))
	switch n, err := r.Read(b); {
	case err != nil:
		panic(err)
	case n != len(b):
		panic("Could not read enough randomness")
	}

	return fmt.Sprintf("%x", b)[:n]
}
