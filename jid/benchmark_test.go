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

func BenchmarkParseString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse("user@example.com/resource")
	}
}

func BenchmarkParseStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse("user@127.0.0.1/resource")
	}
}

func BenchmarkParseStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse("user@[::1]/resource")
	}
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New("user", "example.com", "resource")
	}
}

func BenchmarkCopy(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Copy()
	}
}

func BenchmarkBare(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		j.Bare()
	}
}

func BenchmarkString(b *testing.B) {
	j := &JID{"user", "example.com", "resource"}
	for i := 0; i < b.N; i++ {
		_ = j.String()
	}
}

func BenchmarkEscape(b *testing.B) {
	src := []byte(escape)
	dst := make([]byte, len(src)+18)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Escape.Transform(dst, src, true)
	}
}

func BenchmarkUnescape(b *testing.B) {
	src := []byte(allescaped)
	dst := make([]byte, len(src)/3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Unescape.Transform(dst, src, true)
	}
}
