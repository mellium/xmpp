// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package roster_test

import (
	"context"
	"encoding/xml"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

var (
	_ xml.Marshaler       = (*roster.Item)(nil)
	_ xmlstream.Marshaler = (*roster.Item)(nil)
	_ xmlstream.WriterTo  = (*roster.Item)(nil)
)

var testCases = [...]struct {
	items []roster.Item
	err   error
}{
	0: {},
	1: {
		items: []roster.Item{{
			JID:          jid.MustParse("juliet@example.com"),
			Name:         "Juliet",
			Subscription: "both",
			Group:        []string{"Friends", "Other"},
		}, {
			JID:          jid.MustParse("benvolio@example.org"),
			Name:         "Benvolio",
			Subscription: "to",
		}},
	},
}

func TestFetch(t *testing.T) {
	var IQ = stanza.IQ{ID: "123", Type: stanza.ResultIQ}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			cs := xmpptest.NewClientServer(
				xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					sendIQ := roster.IQ{
						IQ: IQ,
					}
					sendIQ.Query.Item = tc.items
					return e.Encode(sendIQ)
				}),
			)
			iter := roster.FetchIQ(context.Background(), roster.IQ{IQ: IQ}, cs.Client)
			items := make([]roster.Item, 0, len(tc.items))
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

			if !reflect.DeepEqual(items, tc.items) {
				t.Errorf("wrong items:\nwant=\n%+v,\ngot=\n%+v", tc.items, items)
			}
		})
	}
}

func TestFetchNoStart(t *testing.T) {
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			const resp = `<iq id="123" type="result"><query xmlns='jabber:iq:roster'><!-- comment --></query></iq>`
			_, err := xmlstream.Copy(e, xml.NewDecoder(strings.NewReader(resp)))
			return err
		}),
	)
	iter := roster.FetchIQ(context.Background(), roster.IQ{IQ: stanza.IQ{ID: "123"}}, cs.Client)
	for iter.Next() {
		t.Fatalf("iterator should never have any items!")
	}
	if err := iter.Err(); err != nil {
		t.Errorf("Wrong error after iter: want=nil, got=%q", err)
	}
	iter.Close()
}

func TestEmptyIQ(t *testing.T) {
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			const resp = `<iq id="123" type="result"></iq>`
			_, err := xmlstream.Copy(e, xml.NewDecoder(strings.NewReader(resp)))
			return err
		}),
	)
	iter := roster.FetchIQ(context.Background(), roster.IQ{IQ: stanza.IQ{ID: "123"}}, cs.Client)
	for iter.Next() {
		t.Fatalf("iterator should never have any items!")
	}
	if err := iter.Err(); err != nil {
		t.Errorf("Wrong error after iter: want=nil, got=%q", err)
	}
	iter.Close()
}

func TestReceivePush(t *testing.T) {
	const itemJID = "nurse@example.com"
	const x = `<iq xmlns='jabber:client' id='a78b4q6ha463' to='juliet@example.com/chamber' type='set'><query xmlns='jabber:iq:roster' ver='testver'><item jid='` + itemJID + `'/></query></iq>`

	d := xml.NewDecoder(strings.NewReader(x))
	var b strings.Builder
	e := xml.NewEncoder(&b)

	called := false
	h := roster.Handler{
		Push: func(ver string, item roster.Item) error {
			if ver != "testver" {
				t.Errorf("wrong version: want=%q, got=%q", "testver", ver)
			}
			if item.JID.String() != itemJID {
				t.Errorf("unexpected JID: want=%q, got=%q", itemJID, item.JID.String())
			}
			called = true
			return nil
		},
	}

	tok, err := d.Token()
	if err != nil {
		t.Errorf("unexpected error popping start token: %v", err)
	}
	start := tok.(xml.StartElement)
	m := mux.New(roster.Handle(h))
	err = m.HandleXMPP(struct {
		xml.TokenReader
		xmlstream.Encoder
	}{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error in handler: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	if !called {
		t.Errorf("expected push handler to be called")
	}

	out := b.String()
	const expected = `<iq xmlns="jabber:client" type="result" from="juliet@example.com/chamber" id="a78b4q6ha463"></iq>`
	if out != expected {
		t.Errorf("wrong response: want=%q, got=%q", expected, out)
	}
}

func TestReceivePushError(t *testing.T) {
	const itemJID = "nurse@example.com"
	const x = `<iq xmlns='jabber:client' id='a78b4q6ha463' to='juliet@example.com/chamber' type='set'><query xmlns='jabber:iq:roster' ver='testver'><item jid='` + itemJID + `'/></query></iq>`

	d := xml.NewDecoder(strings.NewReader(x))
	var b strings.Builder
	e := xml.NewEncoder(&b)

	h := roster.Handler{
		Push: func(ver string, item roster.Item) error {
			return stanza.Error{Condition: stanza.Forbidden}
		},
	}

	tok, err := d.Token()
	if err != nil {
		t.Errorf("unexpected error popping start token: %v", err)
	}
	start := tok.(xml.StartElement)
	m := mux.New(roster.Handle(h))
	err = m.HandleXMPP(struct {
		xml.TokenReader
		xmlstream.Encoder
	}{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error in handler: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()

	const expected = `<iq xmlns="jabber:client" type="error" from="juliet@example.com/chamber" id="a78b4q6ha463"><error><forbidden xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></forbidden></error></iq>`
	if out != expected {
		t.Errorf("wrong response: want=%q, got=%q", expected, out)
	}
}

type errReadWriter struct{}

func (errReadWriter) Write([]byte) (int, error) {
	return 0, errors.New("called Write on errReadWriter")
}

func (errReadWriter) Read([]byte) (int, error) {
	return 0, errors.New("called Read on errReadWriter")
}

func TestErroredDoesNotPanic(t *testing.T) {
	s := xmpptest.NewSession(0, errReadWriter{})
	iter := roster.Fetch(context.Background(), s)
	if iter.Next() {
		t.Errorf("expected false from call to next")
	}
	if err := iter.Close(); err != nil {
		t.Errorf("got unexpected error closing iter: %v", err)
	}
}

var marshalTests = [...]struct {
	in  interface{}
	out string
}{
	0: {
		in:  roster.IQ{},
		out: `<iq type=""><query xmlns="jabber:iq:roster" ver=""></query></iq>`,
	},
	1: {
		in: roster.IQ{
			Query: struct {
				Ver  string        `xml:"ver,attr"`
				Item []roster.Item `xml:"item"`
			}{
				Ver: "123",
				Item: []roster.Item{
					{},
					{Name: "foo"},
				},
			},
		},
		out: `<iq type=""><query xmlns="jabber:iq:roster" ver="123"><item></item><item name="foo"></item></query></iq>`,
	},
	2: {
		in:  roster.Item{},
		out: `<item></item>`,
	},
	3: {
		in: roster.Item{
			JID:          jid.MustParse("example.net"),
			Name:         "foo",
			Subscription: "sub",
			Group:        []string{"one", "two"},
		},
		out: `<item jid="example.net" name="foo" subscription="sub"><group>one</group><group>two</group></item>`,
	},
}

func TestMarshal(t *testing.T) {
	for i, tc := range marshalTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			b, err := xml.Marshal(tc.in)
			if err != nil {
				t.Fatalf("error marshaling IQ: %v", err)
			}
			if string(b) != tc.out {
				t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, b)
			}
		})
	}
}

func TestUnmarshalItem(t *testing.T) {
	const itemXML = `<item jid="example.net" name="foo" subscription="sub"><group>one</group><group>two</group></item>`
	item := roster.Item{}
	err := xml.Unmarshal([]byte(itemXML), &item)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}
	want := roster.Item{
		JID:          jid.MustParse("example.net"),
		Name:         "foo",
		Subscription: "sub",
		Group:        []string{"one", "two"},
	}
	if !reflect.DeepEqual(want, item) {
		t.Errorf("wrong output: want=%+v, got=%+v", want, item)
	}
}
