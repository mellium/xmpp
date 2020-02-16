// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mux_test

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var (
	passTest = errors.New("mux_test: PASSED")
	failTest = errors.New("mux_test: FAILED")
)

const exampleNS = "com.example"

type passHandler struct{}

func (passHandler) HandleXMPP(xmlstream.TokenReadEncoder, *xml.StartElement) error   { return passTest }
func (passHandler) HandleMessage(stanza.Message, xmlstream.TokenReadEncoder) error   { return passTest }
func (passHandler) HandlePresence(stanza.Presence, xmlstream.TokenReadEncoder) error { return passTest }
func (passHandler) HandleIQ(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return passTest
}

type failHandler struct{}

func (failHandler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return failTest
}
func (failHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	return failTest
}
func (failHandler) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
	return failTest
}
func (failHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return failTest
}

var testCases = [...]struct {
	m           []mux.Option
	x           string
	expectNil   bool
	expectPanic bool
}{
	0: {
		// Basic muxing based on localname and IQ type should work.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, passHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, failHandler{}),
		},
		x: `<iq xml:lang="en-us" type="get" xmlns="jabber:client"></iq>`,
	},
	1: {
		// Basic muxing isn't affected by the server namespace.
		m: []mux.Option{
			mux.IQFunc(stanza.SetIQ, xml.Name{}, passHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, failHandler{}),
		},
		x: `<iq type="set" xmlns="jabber:server"></iq>`,
	},
	2: {
		// The message option works with a client namespace.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.Message(stanza.ChatMessage, passHandler{}),
		},
		x: `<message id="123" type="chat" xmlns="jabber:client"></message>`,
	},
	3: {
		// The message option works with a server namespace.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.MessageFunc(stanza.ChatMessage, passHandler{}.HandleMessage),
		},
		x: `<message to="feste@example.net" from="olivia@example.net" type="chat" xmlns="jabber:server"></message>`,
	},
	4: {
		// The presence option works with a client namespace and no type attribute.
		m: []mux.Option{
			mux.Message(stanza.HeadlineMessage, failHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, passHandler{}),
		},
		x: `<presence id="484" xml:lang="es" xmlns="jabber:client"></presence>`,
	},
	5: {
		m: []mux.Option{
			// The presence option works with a server namespace and an empty type
			// attribute.
			mux.Message(stanza.ChatMessage, failHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.PresenceFunc(stanza.AvailablePresence, passHandler{}.HandlePresence),
		},
		x: `<presence type="" xmlns="jabber:server"></presence>`,
	},
	6: {
		// Other top level elements can be routed with a wildcard namespace.
		m: []mux.Option{mux.Handle(xml.Name{Local: "test"}, passHandler{})},
		x: `<test xmlns="summertime"/>`,
	},
	7: {
		// Other top level elements can be routed with a wildcard localname.
		m: []mux.Option{mux.HandleFunc(xml.Name{Space: "summertime"}, passHandler{}.HandleXMPP)},
		x: `<test xmlns="summertime"/>`,
	},
	8: {
		// Other top level elements can be routed with an exact match.
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "test"}, failHandler{}),
			mux.HandleFunc(xml.Name{Space: "summertime"}, failHandler{}.HandleXMPP),
			mux.HandleFunc(xml.Name{Local: "test", Space: "summertime"}, passHandler{}.HandleXMPP),
		},
		x: `<test xmlns="summertime"/>`,
	},
	9: {
		// IQ exact child match handler.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "a", Space: exampleNS}, failHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{Local: "test", Space: "b"}, failHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{Local: "a", Space: exampleNS}, failHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{Local: "test", Space: "b"}, failHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{Local: "test", Space: exampleNS}, failHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{Local: "test", Space: exampleNS}, passHandler{}),
		},
		x: `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
	},
	10: {
		// If no exact match is available, fallback to the namespace wildcard
		// handler.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "test", Space: ""}, passHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x: `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
	},
	11: {
		// If no exact match or namespace handler is available, fallback local name
		// handler.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, passHandler{}),
			mux.IQ(stanza.ResultIQ, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x: `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
	},
	12: {
		// If no exact match or localname/namespace wildcard is available, fallback
		// to just matching on type alone.
		m: []mux.Option{
			mux.IQ(stanza.ResultIQ, xml.Name{Local: "test", Space: exampleNS}, failHandler{}),
			mux.IQ(stanza.ErrorIQ, xml.Name{}, passHandler{}),
		},
		x: `<iq type="error" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
	},
	13: {
		// Test nop non-stanza handler.
		x:         `<nop/>`,
		expectNil: true,
	},
	14: {
		// Test nop message handler.
		m:         []mux.Option{mux.Message(stanza.HeadlineMessage, failHandler{})},
		x:         `<message xml:lang="de" type="chat" xmlns="jabber:server"/>`,
		expectNil: true,
	},
	15: {
		// Test nop presence handler.
		m:         []mux.Option{mux.Presence(stanza.SubscribedPresence, failHandler{})},
		x:         `<presence to="romeo@example.net" from="mercutio@example.net" xmlns="jabber:server"/>`,
		expectNil: true,
	},
	16: {
		// Expect nil IQ handler to panic.
		m:           []mux.Option{mux.IQ(stanza.GetIQ, xml.Name{}, nil)},
		expectPanic: true,
	},
	17: {
		// Expect nil message handler to panic.
		m:           []mux.Option{mux.Message(stanza.ChatMessage, nil)},
		expectPanic: true,
	},
	18: {
		// Expect nil presence handler to panic.
		m:           []mux.Option{mux.Presence(stanza.ProbePresence, nil)},
		expectPanic: true,
	},
	19: {
		// Expect nil top level element handler to panic.
		m:           []mux.Option{mux.Handle(xml.Name{Local: "test"}, nil)},
		expectPanic: true,
	},
	20: {
		// Expect duplicate IQ handler to panic.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
		},
		expectPanic: true,
	},
	21: {
		// Expect duplicate message handler to panic.
		m: []mux.Option{
			mux.Message(stanza.ChatMessage, failHandler{}),
			mux.Message(stanza.ChatMessage, failHandler{}),
		},
		expectPanic: true,
	},
	22: {
		// Expect duplicate presence handler to panic.
		m: []mux.Option{
			mux.Presence(stanza.ProbePresence, failHandler{}),
			mux.Presence(stanza.ProbePresence, failHandler{}),
		},
		expectPanic: true,
	},
	23: {
		// Expect duplicate top level element handler to panic.
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "test"}, failHandler{}),
			mux.Handle(xml.Name{Local: "test"}, failHandler{}),
		},
		expectPanic: true,
	},
}

type nopEncoder struct {
	xml.TokenReader
}

func (nopEncoder) Encode(interface{}) error                          { return nil }
func (nopEncoder) EncodeElement(interface{}, xml.StartElement) error { return nil }
func (nopEncoder) EncodeToken(xml.Token) error                       { return nil }

func TestMux(t *testing.T) {
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic")
					}
				}()
			}
			m := mux.New(tc.m...)
			d := xml.NewDecoder(strings.NewReader(tc.x))
			tok, _ := d.Token()
			start := tok.(xml.StartElement)

			err := m.HandleXMPP(nopEncoder{TokenReader: d}, &start)
			if (!tc.expectNil && err != passTest) || (tc.expectNil && err != nil) {
				t.Fatalf("unexpected error: `%v'", err)
			}
		})
	}
}

type testEncoder struct {
	xml.TokenReader
	xmlstream.TokenWriter
}

func (e testEncoder) Encode(v interface{}) error {
	return marshal.EncodeXML(e.TokenWriter, v)
}
func (e testEncoder) EncodeElement(v interface{}, start xml.StartElement) error {
	return marshal.EncodeXMLElement(e.TokenWriter, v, start)
}

func TestFallback(t *testing.T) {
	buf := &bytes.Buffer{}
	rw := struct {
		io.Reader
		io.Writer
	}{
		Reader: strings.NewReader(`<iq xmlns="jabber:client" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`),
		Writer: buf,
	}
	s := xmpptest.NewSession(0, rw)

	r := s.TokenReader()
	defer r.Close()
	tok, err := r.Token()
	if err != nil {
		t.Fatalf("Bad start token read: `%v'", err)
	}
	start := tok.(xml.StartElement)
	w := s.TokenWriter()
	defer w.Close()
	err = mux.New().HandleXMPP(testEncoder{
		TokenReader: r,
		TokenWriter: w,
	}, &start)
	if err != nil {
		t.Errorf("Unexpected error: `%v'", err)
	}
	if err := w.Flush(); err != nil {
		t.Errorf("Unexpected error flushing token writer: %q", err)
	}

	const expected = `<iq type="error" to="juliet@example.com" from="romeo@example.com" id="123"><error type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable></error></iq>`
	if buf.String() != expected {
		t.Errorf("Bad output:\nwant=`%v'\n got=`%v'", expected, buf.String())
	}
}

func TestLazyServeMuxMapInitialization(t *testing.T) {
	m := &mux.ServeMux{}

	// This will panic if the map isn't initialized lazily.
	mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{})(m)
}
