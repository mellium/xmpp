// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package receipts_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/stanza"
)

var (
	_ xml.Marshaler         = receipts.Requested{}
	_ xmlstream.Marshaler   = receipts.Requested{}
	_ xmlstream.WriterTo    = receipts.Requested{}
	_ xml.Unmarshaler       = (*receipts.Requested)(nil)
	_ xmlstream.Transformer = receipts.Request
)

var requestTestCases = [...]struct {
	in  string
	out string
}{
	0: {},
	1: {
		in:  `<message xmlns="jabber:client"/>`,
		out: `<message xmlns="jabber:client"><request xmlns="urn:xmpp:receipts"></request></message>`,
	},
	2: {
		in:  `<message xmlns="jabber:server"/><message xmlns="jabber:client"><body>test</body></message>`,
		out: `<message xmlns="jabber:server"><request xmlns="urn:xmpp:receipts"></request></message><message xmlns="jabber:client"><body xmlns="jabber:client">test</body><request xmlns="urn:xmpp:receipts"></request></message>`,
	},
	3: {
		in:  `<message xmlns="jabber:badns"/>`,
		out: `<message xmlns="jabber:badns"></message>`,
	},
	4: {
		in:  `<message xmlns="jabber:client" type="error"/>`,
		out: `<message xmlns="jabber:client" type="error"></message>`,
	},
	5: {
		in:  `<message xmlns="jabber:server" type="error"/>`,
		out: `<message xmlns="jabber:server" type="error"></message>`,
	},
	6: {
		in:  `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt></message>`,
		out: `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt></message>`,
	},
	7: {
		in:  `<message xmlns="jabber:server" type="chat"><body>Test</body><receipt xmlns="urn:xmpp:receipts"></receipt></message>`,
		out: `<message xmlns="jabber:server" type="chat"><body xmlns="jabber:server">Test</body><receipt xmlns="urn:xmpp:receipts"></receipt></message>`,
	},
	8: {
		in:  `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt></message><message xmlns="jabber:client" type="chat"></message>`,
		out: `<message xmlns="jabber:server" type="chat"><receipt xmlns="urn:xmpp:receipts"></receipt></message><message xmlns="jabber:client" type="chat"><request xmlns="urn:xmpp:receipts"></request></message>`,
	},
}

func TestRequest(t *testing.T) {
	for i, tc := range requestTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := receipts.Request(xml.NewDecoder(strings.NewReader(tc.in)))
			// Prevent duplicate xmlns attributes. See https://mellium.im/issue/75
			r = xmlstream.RemoveAttr(func(start xml.StartElement, attr xml.Attr) bool {
				return (start.Name.Local == "message" || start.Name.Local == "iq" || start.Name.Local == "receipt") && attr.Name.Local == "xmlns"
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

func TestMarshal(t *testing.T) {
	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	err := e.Encode(struct {
		stanza.Message

		Requested receipts.Requested
	}{})
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	if err = e.Flush(); err != nil {
		t.Fatalf("error flushing: %v", err)
	}

	const expected = `<message to="" from=""><request xmlns="urn:xmpp:receipts"></request></message>`
	if out := buf.String(); expected != out {
		t.Errorf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

var unmarshalTestCases = [...]struct {
	in  string
	out bool
}{
	0: {
		in:  `<message><request xmlns="urn:xmpp:receipts"/></message>`,
		out: true,
	},
	1: {in: `<message><wrong xmlns="urn:xmpp:receipts"/></message>`},
	2: {in: `<message><request xmlns="urn:xmpp:wrongns"/></message>`},
}

func TestUnmarshal(t *testing.T) {
	for i, tc := range unmarshalTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := struct {
				stanza.Message
				Requested receipts.Requested
			}{}
			err := xml.NewDecoder(strings.NewReader(tc.in)).Decode(&m)
			if err != nil {
				t.Errorf("error decoding: %v", err)
			}
			if m.Requested.Value != tc.out {
				t.Errorf("bad decode: want=%t, got=%t", tc.out, m.Requested.Value)
			}
		})
	}
}

func TestClosedDoesNotPanic(t *testing.T) {
	h := &receipts.Handler{}

	bw := &bytes.Buffer{}
	s := xmpptest.NewClientSession(0, bw)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := h.SendMessageElement(ctx, s, nil, stanza.Message{
		ID: "123",
	})
	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := stanza.Message{
		XMLName: xml.Name{Space: stanza.NSClient, Local: "message"},
		Type:    stanza.ChatMessage,
	}
	r := msg.Wrap(xmlstream.Wrap(nil, xml.StartElement{
		Name: xml.Name{Local: "received", Space: receipts.NS},
		Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "123"}},
	}))

	bw = &bytes.Buffer{}
	e := xml.NewEncoder(bw)
	// If the has not been removed from handling when the context is canceled,
	// this will panic (effectively failing the test).
	err = h.HandleMessage(msg, struct {
		xml.TokenReader
		xmlstream.Encoder
	}{
		TokenReader: r,
		Encoder:     e,
	})
	if err != nil {
		t.Fatalf("error handling response: %v", err)
	}
}

// TODO: find a way to test that SendMessageElement actually matches up the
// response correctly (ie. don't timeout, use the test server).
func TestRoundTrip(t *testing.T) {
	h := &receipts.Handler{}

	var req bytes.Buffer
	s := xmpptest.NewClientSession(0, &req)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := h.SendMessageElement(ctx, s, nil, stanza.Message{
		ID:   "123",
		Type: stanza.NormalMessage,
	})
	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	d := xml.NewDecoder(strings.NewReader(req.String()))
	tok, _ := d.Token()
	start := tok.(xml.StartElement)
	var b strings.Builder
	e := xml.NewEncoder(&b)

	m := mux.New(stanza.NSClient, receipts.Handle(h))
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
	const expected = `<message xmlns="jabber:client" type="normal"><received xmlns="urn:xmpp:receipts" id="123"></received></message>`
	if out != expected {
		t.Errorf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}
