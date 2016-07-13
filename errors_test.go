// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"fmt"
	"testing"
)

var (
	_ error        = (*StanzaError)(nil)
	_ error        = StanzaError{}
	_ fmt.Stringer = (*errorType)(nil)
	_ fmt.Stringer = Auth
)

func TestErrorReturnsCondition(t *testing.T) {
	s := StanzaError{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	s = StanzaError{Condition: "nope", Text: "Text"}
	if s.Text != s.Error() {
		t.Errorf("Expected stanza error to return text `Text` but got %s", s.Error())
	}
}
