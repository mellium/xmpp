// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"testing"
)

func BenchmarkJidFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromString("user@example.com/resource")
	}
}

func BenchmarkJidFromStringUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromStringUnsafe("user@example.com/resource")
	}
}

func BenchmarkJidFromParts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromParts("user", "example.com", "resource")
	}
}

func BenchmarkJidFromPartsUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromPartsUnsafe("user", "example.com", "resource")
	}
}
