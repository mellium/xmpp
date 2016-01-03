// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import "testing"

func BenchmarkRandomIDEven(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RandomID(8)
	}
}

func BenchmarkRandomIDOdd(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RandomID(9)
	}
}

func TestRandomIDLength(t *testing.T) {
	for i := 0; i <= 15; i++ {
		if s := RandomID(i); len(s) != i {
			t.Logf("Expected length %d got %d", i, len(s))
			t.Fail()
		}
	}
}
