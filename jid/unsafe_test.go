// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid_test

import (
	"fmt"
	"testing"

	"mellium.im/xmpp/jid"
)

func TestInvalidUnsafeParseJIDs(t *testing.T) {
	for i, tc := range invalidJIDs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := jid.ParseUnsafe(tc)
			if err != nil {
				t.Errorf("Expected unsafe JID to be valid, got: %q", err)
			}
		})
	}
}

func TestInvalidUnsafeJIDs(t *testing.T) {
	for i, tc := range invalidParts {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic in NewUnsafe: %q", r)
				}
			}()
			u := jid.NewUnsafe(tc.lp, tc.dp, tc.rp)
			if u.Localpart() != tc.lp {
				t.Errorf("Unexpected localpart")
			}
			if u.Domainpart() != tc.dp {
				t.Errorf("Unexpected domainpart")
			}
			if u.Resourcepart() != tc.rp {
				t.Errorf("Unexpected resourcepart")
			}
		})
	}
}
