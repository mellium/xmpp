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
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

var (
	_ xmlstream.WriterTo  = ping.IQ{}
	_ xmlstream.Marshaler = ping.IQ{}
	_ mux.IQHandler       = ping.Handler{}
)

func TestEncode(t *testing.T) {
	j := jid.MustParse("feste@example.net")

	ping := ping.IQ{
		IQ: stanza.IQ{To: j},
	}

	t.Run("marshal", func(t *testing.T) {
		out, err := xml.Marshal(ping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		const expected = `<iq id="" to="feste@example.net" from="" type="get"><ping xmlns="urn:xmpp:ping"></ping></iq>`
		if string(out) != expected {
			t.Errorf("wrong encoding: want=%s, got=%s", expected, out)
		}
	})

	t.Run("write", func(t *testing.T) {
		var b strings.Builder
		e := xml.NewEncoder(&b)
		_, err := ping.WriteXML(e)
		if err != nil {
			t.Fatalf("error writing XML token stream: %v", err)
		}
		err = e.Flush()
		if err != nil {
			t.Fatalf("error flushing token stream: %v", err)
		}

		const expected = `<iq type="" to="feste@example.net"><ping xmlns="urn:xmpp:ping"></ping></iq>`
		if streamOut := b.String(); streamOut != expected {
			t.Errorf("wrong stream encoding: want=%s, got=%s", expected, streamOut)
		}
	})
}

type tokenReadEncoder struct {
	xml.TokenReader
	xmlstream.Encoder
}

func TestRoundTrip(t *testing.T) {
	m := mux.New(ping.Handle())
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

	m := mux.New(mux.IQ(stanza.SetIQ, xml.Name{Local: "ping", Space: ping.NS}, ping.Handler{}))
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

	m := mux.New(mux.IQ(stanza.GetIQ, xml.Name{Local: "badlocal", Space: ping.NS}, ping.Handler{}))
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

	m := mux.New(mux.IQ(stanza.GetIQ, xml.Name{Local: "ping", Space: "badnamespace"}, ping.Handler{}))
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
