// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ping_test

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

var (
	_ xml.Marshaler       = ping.IQ{}
	_ xmlstream.WriterTo  = ping.IQ{}
	_ xmlstream.Marshaler = ping.IQ{}
	_ mux.IQHandler       = ping.Handler{}
	_ info.FeatureIter    = ping.Handler{}
)

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		NoUnmarshal: true,
		Value: &ping.IQ{
			IQ: stanza.IQ{To: jid.MustParse("feste@example.net")},
		},
		XML: `<iq type="" to="feste@example.net"><ping xmlns="urn:xmpp:ping"></ping></iq>`,
	},
	1: {
		Value: &ping.IQ{
			IQ: stanza.IQ{
				Type: stanza.GetIQ,
				To:   jid.MustParse("feste@example.net"),
			},
		},
		XML: `<iq type="get" to="feste@example.net"><ping xmlns="urn:xmpp:ping"></ping></iq>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}

type tokenReadEncoder struct {
	xml.TokenReader
	xmlstream.Encoder
}

func TestRoundTrip(t *testing.T) {
	m := mux.New(stanza.NSClient, ping.Handle())
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandler(m),
	)

	err := ping.Send(context.Background(), cs.Client, cs.Server.LocalAddr())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrongIQType(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	d := xml.NewDecoder(strings.NewReader(`<iq type="set"><ping xmlns="urn:xmpp:ping"/></iq>`))
	tok, _ := d.Token()
	start := tok.(xml.StartElement)

	m := mux.New(stanza.NSClient, mux.IQ(stanza.SetIQ, xml.Name{Local: "ping", Space: ping.NS}, ping.Handler{}))
	err := m.HandleXMPP(tokenReadEncoder{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error handling ping: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()
	if out != "" {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestBadPayloadLocalname(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	d := xml.NewDecoder(strings.NewReader(`<iq type="get"><badlocal xmlns="urn:xmpp:ping"/></iq>`))
	tok, _ := d.Token()
	start := tok.(xml.StartElement)

	m := mux.New(stanza.NSClient, mux.IQ(stanza.GetIQ, xml.Name{Local: "badlocal", Space: ping.NS}, ping.Handler{}))
	err := m.HandleXMPP(tokenReadEncoder{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error handling ping: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()
	if out != "" {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestBadPayloadNamespace(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	d := xml.NewDecoder(strings.NewReader(`<iq type="get"><ping xmlns="badnamespace"/></iq>`))
	tok, _ := d.Token()
	start := tok.(xml.StartElement)

	m := mux.New(stanza.NSClient, mux.IQ(stanza.GetIQ, xml.Name{Local: "ping", Space: "badnamespace"}, ping.Handler{}))
	err := m.HandleXMPP(tokenReadEncoder{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error handling ping: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()
	if out != "" {
		t.Errorf("unexpected output: %s", out)
	}
}
