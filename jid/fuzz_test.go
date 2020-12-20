// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build fuzz

// This tool is meant to excersize JID parsing with random strings.
// This will hopeful tease out any panics that are hidden throughout the code.
// No care has been taken to try and make this fast, so it is very slow and does
// not get run with the normal tests.

package jid_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"mellium.im/xmpp/jid"
)

const (
	// The maximum length of generated JIDs.
	maxLength = 1000

	// The number of JIDs to generate.
	iterations = 1 << 21
)

func randJID(size int) string {
	var b strings.Builder
	for b.Len() < size {
		b.WriteRune(rune(rand.Uint32()))
	}
	return b.String()
}

func TestFuzz(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < iterations; i++ {
		func() {
			randJID := randJID(rand.Intn(maxLength))
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic recovered %v on JID %x", r, randJID)
				}
			}()
			jid.Parse(randJID)
		}()
	}
}
