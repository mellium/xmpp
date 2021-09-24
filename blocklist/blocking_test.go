// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package blocklist_test

import (
	"context"
	"encoding/xml"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/blocklist"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var matchTestCases = [...]struct {
	j1, j2 jid.JID
	result bool
}{
	// Full JID match
	0: {j1: jid.MustParse("user@domain/resource"), j2: jid.MustParse("user@domain/resource"), result: true},
	1: {j1: jid.MustParse("otheruser@domain/resource"), j2: jid.MustParse("user@domain/resource"), result: false},
	2: {j1: jid.MustParse("user@otherdomain/resource"), j2: jid.MustParse("user@domain/resource"), result: false},
	3: {j1: jid.MustParse("user@domain/otherresource"), j2: jid.MustParse("user@domain/resource"), result: false},
	4: {j1: jid.MustParse("otherdomain/resource"), j2: jid.MustParse("user@domain/resource"), result: false},
	5: {j1: jid.MustParse("user@domain"), j2: jid.MustParse("user@domain/resource"), result: false},
	6: {j1: jid.MustParse("domain"), j2: jid.MustParse("user@domain/resource"), result: false},

	// Bare JID match
	7:  {j1: jid.MustParse("user@domain"), j2: jid.MustParse("user@domain"), result: true},
	8:  {j1: jid.MustParse("user@domain/res"), j2: jid.MustParse("user@domain"), result: true},
	9:  {j1: jid.MustParse("domain"), j2: jid.MustParse("user@domain"), result: false},
	10: {j1: jid.MustParse("domain/res"), j2: jid.MustParse("user@domain"), result: false},
	11: {j1: jid.MustParse("otheruser@domain"), j2: jid.MustParse("user@domain"), result: false},

	// Full domain match
	12: {j1: jid.MustParse("domain/resource"), j2: jid.MustParse("domain/resource"), result: true},
	13: {j1: jid.MustParse("user@domain/resource"), j2: jid.MustParse("domain/resource"), result: true},
	14: {j1: jid.MustParse("domain"), j2: jid.MustParse("domain/resource"), result: false},
	15: {j1: jid.MustParse("user@domain"), j2: jid.MustParse("domain/resource"), result: false},
	16: {j1: jid.MustParse("otherdomain/resource"), j2: jid.MustParse("domain/resource"), result: false},
	17: {j1: jid.MustParse("domain/otherresource"), j2: jid.MustParse("domain/resource"), result: false},

	// Bare domain match
	18: {j1: jid.MustParse("domain"), j2: jid.MustParse("domain"), result: true},
	19: {j1: jid.MustParse("domain/resource"), j2: jid.MustParse("domain"), result: true},
	20: {j1: jid.MustParse("user@domain"), j2: jid.MustParse("domain"), result: true},
	21: {j1: jid.MustParse("user@domain/resource"), j2: jid.MustParse("domain"), result: true},
	22: {j1: jid.MustParse("otherdomain"), j2: jid.MustParse("domain"), result: false},
	23: {j1: jid.MustParse("user@otherdomain"), j2: jid.MustParse("domain"), result: false},
	24: {j1: jid.MustParse("user@otherdomain/res"), j2: jid.MustParse("domain"), result: false},
	25: {j1: jid.MustParse("otherdomain/res"), j2: jid.MustParse("domain"), result: false},
}

func TestMatch(t *testing.T) {
	for i, tc := range matchTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := blocklist.Match(tc.j1, tc.j2)
			if tc.result != result {
				t.Errorf("unexpected result: got=%t, want=%t", result, tc.result)
			}
		})
	}
}

var testCases = [...]struct {
	items []jid.JID
	err   error
}{
	0: {},
	1: {
		items: []jid.JID{
			jid.MustParse("juliet@example.com"),
			jid.MustParse("benvolio@example.org"),
		},
	},
}

func TestFetch(t *testing.T) {
	var IQ = stanza.IQ{ID: "123"}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var list []jid.JID
			h := blocklist.Handler{
				Block: func(j jid.JID) {
					list = append(list, j)
				},
				Unblock: func(j jid.JID) {
					b := list[:0]
					for _, x := range list {
						if !j.Equal(x) {
							b = append(b, x)
						}
					}
					list = b
				},
				UnblockAll: func() {
					list = list[:0]
				},
				List: func(j chan<- jid.JID) {
					for _, jj := range list {
						j <- jj
					}
				},
			}
			m := mux.New(ns.Client, blocklist.Handle(h))
			cs := xmpptest.NewClientServer(xmpptest.ServerHandler(m))

			err := blocklist.AddIQ(context.Background(), IQ, cs.Client, tc.items...)
			if err != nil {
				t.Fatalf("error setting the blocklist: %v", err)
			}

			iter := blocklist.FetchIQ(context.Background(), IQ, cs.Client)
			items := make([]jid.JID, 0, len(tc.items))
			for iter.Next() {
				items = append(items, iter.JID())
			}
			if err := iter.Err(); err != tc.err {
				t.Errorf("wrong error after iter: want=%v, got=%v", tc.err, err)
			}
			iter.Close()

			// Don't try to compare nil and empty slice with DeepEqual
			if len(tc.items) == 0 {
				tc.items = make([]jid.JID, 0)
			}

			if !reflect.DeepEqual(items, tc.items) {
				t.Errorf("wrong items:\nwant=\n%+v,\ngot=\n%+v", tc.items, items)
			}

			// Test removing one item.
			if len(tc.items) > 0 {
				err = blocklist.RemoveIQ(context.Background(), IQ, cs.Client, tc.items[0])
				if err != nil {
					t.Errorf("error removing first blocklist item: %v", err)
				}
				if !reflect.DeepEqual(list, tc.items[1:]) {
					t.Errorf("wrong items after removing %s:\nwant=\n%+v,\ngot=\n%+v", tc.items[0], tc.items[1:], list)
				}
			}

			// Test removing all items.
			err = blocklist.RemoveIQ(context.Background(), IQ, cs.Client)
			if err != nil {
				t.Errorf("error removing remaining blocklist items: %v", err)
			}
			if len(list) > 0 {
				t.Errorf("failed to remove remaining items")
			}
		})
	}
}

func TestFetchNoStart(t *testing.T) {
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			const resp = `<iq id="123" type="result"><blocklist xmlns='urn:xmpp:blocklist'><!-- comment --></blocklist></iq>`
			_, err := xmlstream.Copy(e, xml.NewDecoder(strings.NewReader(resp)))
			return err
		}),
	)
	iter := blocklist.FetchIQ(context.Background(), stanza.IQ{ID: "123"}, cs.Client)
	for iter.Next() {
		// Just iterate
	}
	if err := iter.Err(); err != nil {
		t.Errorf("Wrong error after iter: want=nil, got=%q", err)
	}
	iter.Close()
}

type errReadWriter struct{}

func (errReadWriter) Write([]byte) (int, error) {
	return 0, errors.New("called Write on errReadWriter")
}

func (errReadWriter) Read([]byte) (int, error) {
	return 0, errors.New("called Read on errReadWriter")
}

func TestErroredDoesNotPanic(t *testing.T) {
	s := xmpptest.NewClientSession(0, errReadWriter{})
	iter := blocklist.Fetch(context.Background(), s)
	if iter.Next() {
		t.Errorf("expected false from call to next")
	}
	if err := iter.Close(); err != nil {
		t.Errorf("got unexpected error closing iter: %v", err)
	}
}
