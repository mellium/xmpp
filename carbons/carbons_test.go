// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package carbons_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)

func TestEnableDisable(t *testing.T) {
	var out bytes.Buffer
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			e := xml.NewEncoder(&out)
			err := e.EncodeToken(*start)
			if err != nil {
				return err
			}
			_, err = xmlstream.Copy(e, t)
			if err != nil {
				return err
			}
			return e.Flush()
		}),
	)
	err := carbons.EnableIQ(context.Background(), cs.Client, stanza.IQ{
		Type: stanza.GetIQ,
		ID:   "000",
	})
	if !errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable}) {
		t.Fatalf("unexpected error enabling carbons: %v", err)
	}
	err = carbons.DisableIQ(context.Background(), cs.Client, stanza.IQ{
		Type: stanza.GetIQ,
		ID:   "000",
	})
	if !errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable}) {
		t.Fatalf("unexpected error disabling carbons: %v", err)
	}

	output := out.String()
	const expected = `<iq xmlns="jabber:client" xmlns="jabber:client" type="set" id="000"><enable xmlns="urn:xmpp:carbons:2" xmlns="urn:xmpp:carbons:2"></enable></iq><iq xmlns="jabber:client" xmlns="jabber:client" type="set" id="000"><disable xmlns="urn:xmpp:carbons:2" xmlns="urn:xmpp:carbons:2"></disable></iq>`
	if output != expected {
		t.Errorf("wrong XML:\nwant=%s,\n got=%s", expected, output)
	}
}

func TestWrapReceived(t *testing.T) {
	const (
		msg      = `<message xmlns="jabber:client" type="chat" to="romeo@montague.example/garden" from="juliet@capulet.example/balcony"><body>What man art thou that, thus bescreened in night, so stumblest on my counsel?</body><thread>0e3141cd80894871a68e6fe6b1ec56fa</thread></message>`
		expected = `<received xmlns="urn:xmpp:carbons:2"><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-01T00:00:00Z">foo</delay><message xmlns="jabber:client" xmlns="jabber:client" type="chat" to="romeo@montague.example/garden" from="juliet@capulet.example/balcony"><body xmlns="jabber:client">What man art thou that, thus bescreened in night, so stumblest on my counsel?</body><thread xmlns="jabber:client">0e3141cd80894871a68e6fe6b1ec56fa</thread></message></forwarded></received>`
	)

	received := carbons.WrapReceived(delay.Delay{Reason: "foo"}, xml.NewDecoder(strings.NewReader(msg)))

	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, received)
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing: %v", err)
	}

	if out := buf.String(); out != expected {
		t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

func TestWrapSent(t *testing.T) {
	const (
		msg      = `<message xmlns="jabber:client" from="romeo@montague.example/home" to="juliet@capulet.example/balcony" type="chat"><body>Neither, fair saint, if either thee dislike.</body><thread>0e3141cd80894871a68e6fe6b1ec56fa</thread></message>`
		expected = `<sent xmlns="urn:xmpp:carbons:2"><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-01T00:00:00Z">bar</delay><message xmlns="jabber:client" xmlns="jabber:client" from="romeo@montague.example/home" to="juliet@capulet.example/balcony" type="chat"><body xmlns="jabber:client">Neither, fair saint, if either thee dislike.</body><thread xmlns="jabber:client">0e3141cd80894871a68e6fe6b1ec56fa</thread></message></forwarded></sent>`
	)

	sent := carbons.WrapSent(delay.Delay{Reason: "bar"}, xml.NewDecoder(strings.NewReader(msg)))

	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, sent)
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing: %v", err)
	}

	if out := buf.String(); out != expected {
		t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

var unwrapTestCases = [...]struct {
	unwrappedXML string
	reason       string
	inXML        string
	sent         bool
	noDelay      bool
}{
	0: {
		unwrappedXML: `<message xmlns="jabber:client" xmlns="jabber:client"></message>`,
		reason:       "Test",
		inXML:        `<received xmlns='urn:xmpp:carbons:2'><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><message xmlns='jabber:client'></message></forwarded></received>`,
	},
	1: {
		unwrappedXML: `<message xmlns="jabber:client" xmlns="jabber:client"></message>`,
		reason:       "Test",
		inXML:        `<sent xmlns='urn:xmpp:carbons:2'><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><message xmlns='jabber:client'></message></forwarded></sent>`,
		sent:         true,
	},
	2: {
		unwrappedXML: `<message xmlns="jabber:client" xmlns="jabber:client"></message>`,
		inXML:        `<sent xmlns='urn:xmpp:carbons:2'><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><message xmlns='jabber:client'></message></forwarded></sent>`,
		sent:         true,
		noDelay:      true,
	},
}

func TestUnwrap(t *testing.T) {
	for i, tc := range unwrapTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var del *delay.Delay
			if !tc.noDelay {
				del = &delay.Delay{}
			}
			r, sent, err := carbons.Unwrap(del, xml.NewDecoder(strings.NewReader(tc.inXML)))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var buf strings.Builder
			e := xml.NewEncoder(&buf)
			_, err = xmlstream.Copy(e, r)
			if err != nil {
				t.Fatalf("error encoding: %v", err)
			}
			err = e.Flush()
			if err != nil {
				t.Fatalf("error flushing: %v", err)
			}
			if out := buf.String(); out != tc.unwrappedXML {
				t.Errorf("wrong XML: want=%v, got=%v", tc.unwrappedXML, out)
			}
			if del != nil && del.Reason != tc.reason {
				t.Errorf("did not unmarshal delay: want=%v, got=%v", tc.reason, del.Reason)
			}
			if sent != tc.sent {
				t.Errorf("wrong cc type: want=%v, got=%v", tc.sent, sent)
			}
		})
	}
}
