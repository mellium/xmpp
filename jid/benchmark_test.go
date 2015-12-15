// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"testing"
)

func BenchmarkSplit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SplitString("user@example.com/resource")
	}
}

func BenchmarkUnsafeFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnsafeFromString("user@example.com/resource")
	}
}

func BenchmarkUnsafeFromStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnsafeFromString("user@127.0.0.1/resource")
	}
}

func BenchmarkUnsafeFromStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnsafeFromString("user@[::1]/resource")
	}
}

func BenchmarkUnsafeFromParts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnsafeFromParts("user", "example.com", "resource")
	}
}

func BenchmarkCopy(b *testing.B) {
	j := &UnsafeJID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Copy()
	}
}

func BenchmarkBare(b *testing.B) {
	j := &UnsafeJID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Bare()
	}
}

func BenchmarkString(b *testing.B) {
	j := &UnsafeJID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.String()
	}
}
