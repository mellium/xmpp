// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"crypto/rand"
	"fmt"
)

const IDLen = 16

// TODO: This will be called a lot, and probably needs to be faster than we can
//       get when reading from getrandom(2). Should we use a fast userspace
//       CSPRNG and just seed with data from the OS?

// RandomID generates a new random identifier of the given length. If the OS's
// entropy pool isn't initialized, or we can't generate random numbers for some
// other reason, panic.
func RandomID(n int) string {
	b := make([]byte, (n/2)+(n&1))
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)[:n]
}
