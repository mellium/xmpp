// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"testing"

	"encoding/xml"
)

func BenchmarkJidSplit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		partsFromString("user@example.com/resource")
	}
}

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

func BenchmarkJidFromJidValidated(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", true}
	for i := 0; i < b.N; i++ {
		FromJid(j)
	}
}

func BenchmarkJidFromJidUnvalidated(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		FromJid(j)
	}
}

func BenchmarkJidCopy(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.Copy()
	}
}

func BenchmarkJidBare(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.Bare()
	}
}

func BenchmarkJidString(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	for i := 0; i < b.N; i++ {
		j.String()
	}
}

func BenchmarkJidMarshalXMLAttr(b *testing.B) {
	j := &Jid{"user", "example.com", "resource", false}
	n := xml.Name{}
	for i := 0; i < b.N; i++ {
		j.MarshalXMLAttr(n)
	}
}
