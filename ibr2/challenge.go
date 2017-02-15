// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package ibr2

// Challenge is an IBR challenge.
// API WARNING: The challenge struct is not complete or usable yet.
type Challenge struct {
	// Type is the type of the challenge as it appears in the server advertised
	// challenges list.
	Type string
}
