// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest_test

import (
	"bytes"
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
)

func TestNewSession(t *testing.T) {
	state := xmpp.Secure | xmpp.InputStreamClosed
	buf := new(bytes.Buffer)
	s := xmpptest.NewSession(state, buf)

	if mask := s.State(); mask != state|xmpp.Ready {
		t.Errorf("Got invalid state value: want=%d, got=%d", state, mask)
	}

	if out := buf.String(); out != "" {
		t.Errorf("Buffer wrote unexpected tokens: `%s'", out)
	}
}
