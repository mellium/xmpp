// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package s2s_test

import (
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/s2s"
)

var bidiTestCases = [...]xmpptest.FeatureTestCase{
	0: {
		State:   xmpp.Received,
		Feature: s2s.Bidi(),
		In:      `<bidi xmlns="urn:xmpp:bidi"></bidi>`,
	},
	1: {
		Feature: s2s.Bidi(),
		Out:     `<bidi xmlns="urn:xmpp:bidi"></bidi>`,
	},
}

func TestBidi(t *testing.T) {
	xmpptest.RunFeatureTests(t, bidiTestCases[:])
}
