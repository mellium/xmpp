// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build gofuzzbeta

package jid_test

import (
	"testing"

	"mellium.im/xmpp/jid"
)

func FuzzParseJID(f *testing.F) {
	f.Add("@")
	f.Add("/")
	f.Add("xn--")
	f.Add("test@example.net")
	f.Fuzz(func(t *testing.T, j string) {
		parsed, err := jid.Parse(j)
		if err != nil {
			t.Skip()
		}
		s := parsed.String()
		parsed2, err := jid.Parse(s)
		if err != nil {
			t.Fatalf("failed to parse a JID that encodes successfully: %q", s)
		}
		if !parsed.Equal(parsed2) {
			t.Errorf("JID parsing/encoding is unstable: %q, %q", parsed, parsed2)
		}
	})
}
