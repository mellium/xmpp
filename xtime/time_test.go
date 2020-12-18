// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xtime_test

import (
	"context"
	"encoding/xml"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/xtime"
)

var (
	_ xml.Marshaler       = xtime.Time{}
	_ xml.Unmarshaler     = (*xtime.Time)(nil)
	_ xmlstream.Marshaler = xtime.Time{}
	_ xmlstream.WriterTo  = xtime.Time{}
)

type tokenReadEncoder struct {
	xml.TokenReader
	xmlstream.Encoder
}

func TestRoundTrip(t *testing.T) {
	serverTime := time.Time{}
	serverTime = serverTime.Add(24 * time.Hour * 7 * 52)
	h := xtime.Handler{
		TimeFunc: func() time.Time {
			return serverTime
		},
	}
	m := mux.New(xtime.Handle(h))
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandler(m),
	)

	respTime, err := xtime.Get(context.Background(), cs.Client, cs.Server.LocalAddr())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !serverTime.Equal(respTime) {
		t.Errorf("wrong time: want=%v, got=%v", serverTime, respTime)
	}
}
