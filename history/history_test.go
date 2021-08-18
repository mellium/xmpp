// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history_test

import (
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/paging"
	"mellium.im/xmpp/stanza"
)

func TestRoundTrip(t *testing.T) {
	count := uint64(10)
	out := paging.Set{Count: &count}
	var foundMessages int
	h := history.NewHandler(mux.MessageHandlerFunc(func(stanza.Message, xmlstream.TokenReadEncoder) error {
		foundMessages++
		return nil
	}))
	m := mux.New(
		mux.IQFunc(stanza.SetIQ, xml.Name{Space: history.NS, Local: "query"}, func(iq stanza.IQ, e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			xmlstream.Copy(e, stanza.Message{}.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: history.NS, Local: "result"}, Attr: []xml.Attr{{
					Name:  xml.Name{Local: "queryid"},
					Value: "123",
				}}},
			)))
			xmlstream.Copy(e, stanza.Message{}.Wrap(xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Space: history.NS, Local: "result"}, Attr: []xml.Attr{{
					Name:  xml.Name{Local: "queryid"},
					Value: "123",
				}}},
			)))
			_, err := xmlstream.Copy(e, iq.Result(xmlstream.Wrap(
				out.TokenReader(),
				xml.StartElement{Name: xml.Name{Space: history.NS, Local: "fin"}},
			)))
			return err
		}),
		history.Handle(h),
	)
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandler(m),
		xmpptest.ClientHandler(m),
	)

	to := jid.MustParse("example.net")
	res, err := history.Fetch(context.Background(), history.Query{AfterID: "123"}, to, cs.Client)
	if err != nil {
		t.Fatalf("error fetching history: %v", err)
	}

	if res.Set.Count == nil || out.Count == nil {
		t.Fatal("did not expect nil count")
	}
	if *res.Set.Count != *out.Count {
		t.Errorf("wrong output: want=%d, got=%d", *out.Count, *res.Set.Count)
	}
	if foundMessages != 2 {
		t.Errorf("expected %d messages, got=%d", 2, foundMessages)
	}

	foundMessages = 0
	iter := h.Fetch(context.Background(), history.Query{ID: "123", AfterID: "123"}, to, cs.Client)
	for iter.Next() {
		if cur := iter.Current(); cur == nil {
			t.Errorf("found nil current message")
		}
		foundMessages++
	}
	if err := iter.Err(); err != nil {
		t.Errorf("error iterating: %v", err)
	}
	if foundMessages != 2 {
		t.Errorf("expected %d messages, got=%d", 2, foundMessages)
	}
	res = iter.Result()
	if res.Set.Count == nil {
		t.Errorf("got nil count")
	} else {
		if *res.Set.Count != 10 {
			t.Errorf("wrong count: want=10, got=%d", *res.Set.Count)
		}
	}

	// This has already been done, but make sure it doesn't panic.
	err = iter.Close()
	if err != nil {
		t.Fatalf("noop close returned error somehow: %v", err)
	}
}
