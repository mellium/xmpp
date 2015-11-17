// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"testing"

	"encoding/xml"
)

func BenchmarkSplit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		partsFromString("user@example.com/resource")
	}
}

func BenchmarkFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromString("user@example.com/resource")
	}
}

func BenchmarkFromStringIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromString("user@127.0.0.1/resource")
	}
}

func BenchmarkFromStringIPv6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromString("user@[::1]/resource")
	}
}

func BenchmarkFromStringUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromStringUnsafe("user@example.com/resource")
	}
}

func BenchmarkFromParts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromParts("user", "example.com", "resource")
	}
}

func BenchmarkFromPartsUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromPartsUnsafe("user", "example.com", "resource")
	}
}

func BenchmarkFromJidValidated(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", true}
	for i := 0; i < b.N; i++ {
		FromJid(j)
	}
}

func BenchmarkFromJidUnvalidated(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		FromJid(j)
	}
}

func BenchmarkCopy(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.Copy()
	}
}

func BenchmarkBare(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.Bare()
	}
}

func BenchmarkString(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.String()
	}
}

func BenchmarkMarshalXMLAttr(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	n := xml.Name{}
	for i := 0; i < b.N; i++ {
		j.MarshalXMLAttr(n)
	}
}
