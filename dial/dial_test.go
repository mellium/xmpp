// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial

import (
	"strconv"
	"testing"

	"mellium.im/xmpp/jid"
)

func TestDialClientPanicsIfNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Dial to panic when passed a nil context.")
		}
	}()
	Client(nil, "tcp", jid.MustParse("feste@shakespeare.lit"))
}

var connTypeTests = [...]struct {
	useTLS bool
	s2s    bool
	svc    string
}{
	0: {useTLS: true, s2s: true, svc: "xmpps-server"},
	1: {useTLS: true, s2s: false, svc: "xmpps-client"},
	2: {useTLS: false, s2s: true, svc: "xmpp-server"},
	3: {useTLS: false, s2s: false, svc: "xmpp-client"},
}

func TestConnType(t *testing.T) {
	for i, tc := range connTypeTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			svc := connType(tc.useTLS, tc.s2s)
			if svc != tc.svc {
				t.Errorf("Wrong conntype value: want=%q, got=%q", tc.svc, svc)
			}
		})
	}
}
