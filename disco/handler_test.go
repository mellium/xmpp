// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco_test

import (
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestFeaturesRoundTrip(t *testing.T) {
	m := mux.New(stanza.NSClient, disco.Handle())
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

type itemHandler struct{}

func (itemHandler) HandleXMPP(xmlstream.TokenReadEncoder, *xml.StartElement) error {
	panic("should not be called")
}

func (itemHandler) ForItems(node string, f func(items.Item) error) error {
	if node != "" {
		return nil
	}

	return f(items.Item{
		Name: disco.NSItems,
	})
}

func TestItemsRoundTrip(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		disco.Handle(),
		mux.Handle(xml.Name{}, itemHandler{}),
	)
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandler(m),
	)

	iter := disco.FetchItemsIQ(context.Background(), "", stanza.IQ{ID: "123"}, cs.Client)
	allItems := []items.Item{}
	for iter.Next() {
		allItems = append(allItems, iter.Item())
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("error iterating over items: %v", err)
	}
	if len(allItems) != 1 || allItems[0].Name != disco.NSItems {
		t.Errorf("wrong items: want=%s, got=%v", disco.NSItems, allItems)
	}
	err := iter.Close()
	if err != nil {
		t.Fatalf("error closing iterator: %v", err)
	}

	iter = disco.FetchItemsIQ(context.Background(), "node", stanza.IQ{ID: "123"}, cs.Client)
	for iter.Next() {
		t.Fatalf("error, got item %v but did not expect any", iter.Item())
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("error iterating over empty items: %v", err)
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing empty iter: %v", err)
	}
}
