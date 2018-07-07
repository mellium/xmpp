// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

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
		_, _ = Parse("user@example.com/resource")
	}
}

func BenchmarkParseStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse("user@127.0.0.1/resource")
	}
}

func BenchmarkParseStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse("user@[::1]/resource")
	}
}

func BenchmarkParseUnsafeString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseUnsafe("user@example.com/resource")
	}
}

func BenchmarkParseUnsafeStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseUnsafe("user@127.0.0.1/resource")
	}
}

func BenchmarkParseUnsafeStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseUnsafe("user@[::1]/resource")
	}
}

func BenchmarkNewFull(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("user", "example.com", "resource")
	}
}

func BenchmarkNewBare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("user", "example.com", "")
	}
}

func BenchmarkNewDomain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("", "example.com", "")
	}
}

func BenchmarkWithResource(b *testing.B) {
	j := MustParse("example.com/res")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = j.WithResource("res")
	}
}

func BenchmarkCopy(b *testing.B) {
	j := JID{4, 11, []byte("userexample.comresource")}
	for i := 0; i < b.N; i++ {
		_ = j.Copy()
	}
}

func BenchmarkBare(b *testing.B) {
	j := JID{4, 11, []byte("userexample.comresource")}
	for i := 0; i < b.N; i++ {
		_ = j.Bare()
	}
}

func BenchmarkString(b *testing.B) {
	j := JID{4, 11, []byte("userexample.comresource")}
	for i := 0; i < b.N; i++ {
		_ = j.String()
	}
}

func BenchmarkEscapeTransform(b *testing.B) {
	src := []byte(escape)
	dst := make([]byte, len(src)+18)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Escape.Transform(dst, src, true)
	}
}

func BenchmarkUnescapeTransform(b *testing.B) {
	src := []byte(allescaped)
	dst := make([]byte, len(src)/3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Unescape.Transform(dst, src, true)
	}
}

func BenchmarkEscapeBytes(b *testing.B) {
	src := []byte(escape)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Escape.Bytes(src)
	}
}

func BenchmarkUnescapeBytes(b *testing.B) {
	src := []byte(allescaped)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unescape.Bytes(src)
	}
}
