// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest_test

import (
	"io"
	"testing"

	"mellium.im/xmpp/internal/xmpptest"
)

func TestTokens(t *testing.T) {
	toks := xmpptest.Tokens{0, 1, 2}
	for i := 0; i < 3; i++ {
		tok, err := toks.Token()
		if err != nil {
			t.Errorf("unexpected error on token %d: %v", i, err)
		}
		if tok.(int) != i {
			t.Errorf("unexpcted token: want=%d, got=%v", i, tok)
		}
	}

	tok, err := toks.Token()
	if err != io.EOF {
		t.Errorf("unexpected error: want=%v, got=%v", io.EOF, err)
	}
	if tok != nil {
		t.Errorf("unexpcted token: %v", tok)
	}
}
