// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package pubsub_test

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"strconv"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/ejabberd"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/pubsub"
)

func TestIntegrationPubFetch(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.Component("pubsub.localhost", "", "pubsub"),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationPubFetch)

	ejabberdRun := ejabberd.Test(context.TODO(), t,
		integration.Log(),
		ejabberd.ListenC2S(),
	)
	ejabberdRun(integrationPubFetch)
}

func integrationPubFetch(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
	session, err := cmd.DialClient(ctx, j, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", pass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error connecting: %v", err)
	}
	go func() {
		err := session.Serve(nil)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()
	for i := 0; i < 10; i++ {
		strID := strconv.Itoa(i)
		newID, err := pubsub.Publish(ctx, session, t.Name(), strID, xmlstream.Wrap(
			nil,
			xml.StartElement{
				Name: xml.Name{Local: "foo"},
				Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: strID}},
			},
		))
		if err != nil {
			t.Fatalf("error publishing %d: %v", i, err)
		}
		if newID != strID {
			t.Errorf("wrong ID for published value: want=%q, got=%q", strID, newID)
		}
	}
	iter := pubsub.Fetch(ctx, session, pubsub.Query{
		Node: t.Name(),
	})

	const strID = "9"
	hasNext := iter.Next()
	if !hasNext {
		t.Fatalf("no item found")
	}
	id, r := iter.Item()
	if id != strID {
		t.Errorf("wrong ID for fetched value: want=%q, got=%q", strID, id)
	}
	foo := struct {
		XMLName xml.Name `xml:"foo"`
		ID      int      `xml:"id,attr"`
	}{}
	err = xml.NewTokenDecoder(r).Decode(&foo)
	if err != nil {
		t.Fatalf("error decoding %s foo: %v", id, err)
	}
	if foo.ID != 9 {
		t.Errorf("wrong value for ID in element: want=%s, got=%d", strID, foo.ID)
	}
	hasNext = iter.Next()
	if hasNext {
		t.Fatalf("expected default pep to only store one item")
	}

	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing iter: %v", err)
	}

	err = pubsub.Delete(ctx, session, t.Name(), "9", false)
	if err != nil {
		t.Fatalf("error retracting pubsub item: %v", err)
	}

	iter = pubsub.Fetch(ctx, session, pubsub.Query{
		Node: t.Name(),
	})
	if iter.Next() {
		t.Fatalf("expected node to be empty after deletion")
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing second iter: %v", err)
	}
}
