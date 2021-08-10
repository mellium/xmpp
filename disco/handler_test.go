// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco_test

import (
	"context"
	"testing"

	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestRoundTrip(t *testing.T) {
	m := mux.New(disco.Handle())
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandler(m),
	)

	info, err := disco.GetInfoIQ(context.Background(), "", stanza.IQ{ID: "123"}, cs.Client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(info.Features) != 1 || info.Features[0].Var != disco.NSInfo {
		t.Errorf("wrong features: want=%s, got=%v", disco.NSInfo, info.Features)
	}

	info, err = disco.GetInfoIQ(context.Background(), "node", stanza.IQ{ID: "123"}, cs.Client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(info.Features) != 0 {
		t.Errorf("got unexpected features %v", info.Features)
	}
}
