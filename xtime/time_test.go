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
	_ xml.MarshalerAttr   = xtime.Time{}
	_ xml.UnmarshalerAttr = (*xtime.Time)(nil)
)

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

func TestAttrMarshal(t *testing.T) {
	zeroTime := time.Time{}.Add(24 * time.Hour)
	xt := xtime.Time{Time: zeroTime}
	name := xml.Name{Space: "example.net", Local: "foo"}
	attr, err := xt.MarshalXMLAttr(name)
	if err != nil {
		t.Fatalf("unexpected error marshaling attr: %v", err)
	}
	if attr.Name != name {
		t.Fatalf("wrong name for attr: want=%v, got=%v", name, attr.Name)
	}
	const expected = "0001-01-02T00:00:00Z"
	if attr.Value != expected {
		t.Fatalf("wrong value for attr: want=%v, got=%v", expected, attr.Value)
	}

	newTime := &xtime.Time{}
	err = newTime.UnmarshalXMLAttr(attr)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling attr: %v", err)
	}
	if *newTime != xt {
		t.Fatalf("times don't match: want=%v, got=%v", xt, newTime)
	}
}
