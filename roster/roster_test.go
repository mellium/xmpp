// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package roster_test

import (
	"context"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

var testCases = [...]struct {
	reply string
	items []roster.Item
	err   error
}{
	0: {
		reply: `<query xmlns='jabber:iq:roster'/>`,
	},
	1: {
		reply: `<query xmlns='jabber:iq:roster'><item jid='juliet@example.com' name='Juliet' subscription='both'><group>Friends</group></item><item jid='benvolio@example.org' name='Benvolio' subscription='to'/></query>`,
		items: []roster.Item{{
			JID:          jid.MustParse("juliet@example.com"),
			Name:         "Juliet",
			Subscription: "both",
			Group:        "Friends",
		}, {
			JID:          jid.MustParse("benvolio@example.org"),
			Name:         "Benvolio",
			Subscription: "to",
		}},
	},
}

func TestFetch(t *testing.T) {
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pr, pw := io.Pipe()
			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: pr,
				// TODO: verify that we write a valid request.
				Writer: ioutil.Discard,
			}
			s := xmpptest.NewSession(0, rw)
			// Start serving the session.
			go func() {
				if err := s.Serve(nil); err != nil {
					panic(err)
				}
			}()
			// The remote server sends a reply.
			go func() {
				_, err := io.Copy(pw, strings.NewReader(`<iq id="123">`+tc.reply+`</iq>`))
				if err != nil {
					panic(err)
				}
			}()
			iter := roster.FetchIQ(context.Background(), stanza.IQ{ID: "123"}, s)
			items := make([]roster.Item, 0, len(tc.items))
			for iter.Next() {
				items = append(items, iter.Item())
			}
			if err := iter.Err(); err != tc.err {
				t.Errorf("Wrong error after iter: want=%q, got=%q", tc.err, err)
			}
			iter.Close()

			// Don't try to compare nil and empty slice with DeepEqual
			if len(items) == 0 && len(tc.items) == 0 {
				return
			}

			if !reflect.DeepEqual(items, tc.items) {
				t.Errorf("Wrong items:\nwant=\n%+v,\ngot=\n%+v", tc.items, items)
			}
		})
	}
}
