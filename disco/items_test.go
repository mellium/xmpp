// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco_test

import (
	"context"
	"encoding/xml"
	"reflect"
	"strconv"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var testFetchItems = [...]struct {
	node  string
	items map[string][]disco.Item
	err   error
}{
	0: {},
	1: {
		items: map[string][]disco.Item{
			"": {{
				XMLName: xml.Name{Space: disco.NSItems, Local: "item"},
				JID:     jid.MustParse("juliet@example.com"),
				Name:    "Juliet",
			}, {
				XMLName: xml.Name{Space: disco.NSItems, Local: "item"},
				JID:     jid.MustParse("benvolio@example.org"),
			}},
		},
	},
	2: {
		node: "test",
		items: map[string][]disco.Item{
			"test": {{
				XMLName: xml.Name{Space: disco.NSItems, Local: "item"},
				JID:     jid.MustParse("benvolio@example.org"),
			}},
		},
	},
}

type queryItems struct {
	XMLName xml.Name     `xml:"http://jabber.org/protocol/disco#items query"`
	Items   []disco.Item `xml:"item"`
}

func TestFetchItems(t *testing.T) {
	var IQ = stanza.IQ{
		ID:   "123",
		Type: stanza.ResultIQ,
	}
	for i, tc := range testFetchItems {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			cs := xmpptest.NewClientServer(
				xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					query := disco.ItemsQuery{}
					err := xml.NewTokenDecoder(e).Decode(&query)
					if err != nil {
						return err
					}
					sendIQ := struct {
						stanza.IQ
						Query queryItems
					}{
						IQ: IQ,
						Query: queryItems{
							Items: tc.items[query.Node],
						},
					}
					return e.Encode(sendIQ)
				}),
			)
			iter := disco.FetchItemsIQ(context.Background(), tc.node, IQ, cs.Client)
			items := make([]disco.Item, 0, len(tc.items))
			for iter.Next() {
				items = append(items, iter.Item())
			}
			if err := iter.Err(); err != tc.err {
				t.Errorf("wrong error after iter: want=%q, got=%q", tc.err, err)
			}
			iter.Close()

			// Don't try to compare nil and empty slice with DeepEqual
			if len(items) == 0 && len(tc.items) == 0 {
				return
			}

			if !reflect.DeepEqual(items, tc.items[tc.node]) {
				t.Errorf("wrong items:\nwant=\n%+v,\ngot=\n%+v", tc.items[tc.node], items)
			}
		})
	}
}
