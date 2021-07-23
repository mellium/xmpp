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
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)
var disableCarbonTestCases = [...]struct {
	in  string
	out string
}{
	0: {},
	1: {
		in:  `<message xmlns="jabber:client"/>`,
		out: `<message xmlns="jabber:client"><private xmlns="urn:xmpp:carbons:2"></private></message>`,
	},
	2: {
		in:  `<message xmlns="jabber:client" type="error"/>`,
		out: `<message xmlns="jabber:client" type="error"><private xmlns="urn:xmpp:carbons:2"></private></message>`,
	},
	3: {
		in : `<message xmlns="jabber:client" type="error"><private xmlns="urn:xmpp:carbons:2"/></message>`,
		out : `<message xmlns="jabber:client" type="error"><private xmlns="urn:xmpp:carbons:2"></private></message>`,
	},
	4: {
		in:  `<message xmlns="jabber:badns"/>`,
		out: `<message xmlns="jabber:badns"></message>`,
	},
	5: {
		in:  `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt></message>`,
		out: `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt><private xmlns="urn:xmpp:carbons:2"></private></message>`,
	},
	6: {
		in:  `<message xmlns="jabber:server" type="error"/>`,
		out: `<message xmlns="jabber:server" type="error"><private xmlns="urn:xmpp:carbons:2"></private></message>`,

	},
}

func TestDisableCarbon(t *testing.T) {
	for i, tc := range disableCarbonTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := carbons.Exclude(xml.NewDecoder(strings.NewReader(tc.in)))
			// Prevent duplicate xmlns attributes. See https://mellium.im/issue/75
			r = xmlstream.RemoveAttr(func(start xml.StartElement, attr xml.Attr) bool {
				return (start.Name.Local == "message" || start.Name.Local == "iq" || start.Name.Local == "private" || start.Name.Local == "receipt") && attr.Name.Local == "xmlns"
			})(r)
			var buf strings.Builder
			e := xml.NewEncoder(&buf)
			_, err := xmlstream.Copy(e, r)
			if err != nil {
				t.Fatalf("error encoding: %v", err)
			}
			if err = e.Flush(); err != nil {
				t.Fatalf("error flushing: %v", err)
			}

			if out := buf.String(); tc.out != out {
				t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, out)
			}
		})
	}
}


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
