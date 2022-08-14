// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mux_test

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var (
	_ info.FeatureIter  = (*mux.ServeMux)(nil)
	_ info.IdentityIter = (*mux.ServeMux)(nil)
	_ items.Iter        = (*mux.ServeMux)(nil)
)

var (
	errPassTest = errors.New("mux_test: PASSED")
	errFailTest = errors.New("mux_test: FAILED")
)

const exampleNS = "com.example"

type passHandler struct{}

func (passHandler) HandleXMPP(xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return errPassTest
}
func (passHandler) HandleMessage(stanza.Message, xmlstream.TokenReadEncoder) error {
	return errPassTest
}
func (passHandler) HandlePresence(stanza.Presence, xmlstream.TokenReadEncoder) error {
	return errPassTest
}
func (passHandler) HandleIQ(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return errPassTest
}

type multiHandler struct{}

func (multiHandler) HandlePresence(_ stanza.Presence, t xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(t)
	data := struct {
		stanza.Presence

		Test    string `xml:"com.example test"`
		Example string `xml:"com.example example"`
	}{}
	err := d.Decode(&data)
	if err != nil {
		return err
	}
	if data.Test != "test" {
		return fmt.Errorf("wrong value for test element: want=%q, got=%q", "test", data.Test)
	}
	if data.Example != "example" {
		return fmt.Errorf("wrong value for example element: want=%q, got=%q", "example", data.Test)
	}
	return errPassTest
}

func (m multiHandler) HandleMessage(_ stanza.Message, t xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(t)
	data := struct {
		stanza.Message

		Test    string `xml:"com.example test"`
		Example string `xml:"com.example example"`
	}{}
	err := d.Decode(&data)
	if err != nil {
		return err
	}
	if data.Test != "test" {
		return fmt.Errorf("wrong value for test element: want=%q, got=%q", "test", data.Test)
	}
	if data.Example != "example" {
		return fmt.Errorf("wrong value for example element: want=%q, got=%q", "example", data.Test)
	}
	return errPassTest
}

type failHandler struct{}

func (failHandler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return errFailTest
}
func (failHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	return errFailTest
}
func (failHandler) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
	return errFailTest
}
func (failHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return errFailTest
}

type decodeHandler struct {
	Body string
}

func (h decodeHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(t)
	data := struct {
		stanza.Message

		Body string `xml:"body"`
	}{}
	err := d.Decode(&data)
	if err != nil {
		return err
	}
	if h.Body != data.Body {
		return fmt.Errorf("wrong body: want=%q, got=%q", h.Body, data.Body)
	}
	return errPassTest
}

var testCases = [...]struct {
	m           []mux.Option
	x           string
	expectPanic bool
	err         error
	stanzaNS    string
}{
	0: {
		// Basic muxing based on localname and IQ type should work.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, passHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, xml.Name{}, failHandler{}),
		},
		x:   `<iq xml:lang="en-us" type="get" xmlns="jabber:client"><a/></iq>`,
		err: errPassTest,
	},
	1: {
		// Basic muxing isn't affected by the server namespace.
		m: []mux.Option{
			mux.IQFunc(stanza.SetIQ, xml.Name{}, passHandler{}.HandleIQ),
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, xml.Name{}, failHandler{}),
		},
		x:        `<iq type="set" xmlns="jabber:server"><b/></iq>`,
		err:      errPassTest,
		stanzaNS: stanza.NSServer,
	},
	2: {
		// The message option works with a client namespace.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.Message(stanza.ChatMessage, xml.Name{}, passHandler{}),
		},
		x:   `<message id="123" type="chat" xmlns="jabber:client"></message>`,
		err: errPassTest,
	},
	3: {
		// The message option works with a server namespace.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.MessageFunc(stanza.ChatMessage, xml.Name{}, passHandler{}.HandleMessage),
		},
		x:        `<message to="feste@example.net" from="olivia@example.net" type="chat" xmlns="jabber:server"></message>`,
		err:      errPassTest,
		stanzaNS: stanza.NSServer,
	},
	4: {
		// The presence option works with a client namespace and no type attribute.
		m: []mux.Option{
			mux.Message(stanza.HeadlineMessage, xml.Name{}, failHandler{}),
			mux.IQ(stanza.SetIQ, xml.Name{}, failHandler{}),
			mux.Presence(stanza.AvailablePresence, xml.Name{}, passHandler{}),
		},
		x:   `<presence id="484" xml:lang="es" xmlns="jabber:client"></presence>`,
		err: errPassTest,
	},
	5: {
		m: []mux.Option{
			// The presence option works with a server namespace and an empty type
			// attribute.
			mux.Message(stanza.ChatMessage, xml.Name{}, failHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
			mux.PresenceFunc(stanza.AvailablePresence, xml.Name{}, passHandler{}.HandlePresence),
		},
		x:        `<presence type="" xmlns="jabber:server"></presence>`,
		err:      errPassTest,
		stanzaNS: stanza.NSServer,
	},
	6: {
		// Other top level elements can be routed with a wildcard namespace.
		m:   []mux.Option{mux.Handle(xml.Name{Local: "test"}, passHandler{})},
		x:   `<test xmlns="summertime"/>`,
		err: errPassTest,
	},
	7: {
		// Other top level elements can be routed with a wildcard localname.
		m:   []mux.Option{mux.HandleFunc(xml.Name{Space: "summertime"}, passHandler{}.HandleXMPP)},
		x:   `<test xmlns="summertime"/>`,
		err: errPassTest,
	},
	8: {
		// Other top level elements can be routed with an exact match.
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "test"}, failHandler{}),
			mux.HandleFunc(xml.Name{Space: "summertime"}, failHandler{}.HandleXMPP),
			mux.HandleFunc(xml.Name{Local: "test", Space: "summertime"}, passHandler{}.HandleXMPP),
		},
		x:   `<test xmlns="summertime"/>`,
		err: errPassTest,
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
		x:   `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
		err: errPassTest,
	},
	10: {
		// If no exact match is available, fallback to the namespace wildcard
		// handler.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "test", Space: ""}, passHandler{}),
			mux.IQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
		err: errPassTest,
	},
	11: {
		// If no exact match or namespace handler is available, fallback local name
		// handler.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, passHandler{}),
			mux.IQ(stanza.ResultIQ, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<iq type="get" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
		err: errPassTest,
	},
	12: {
		// If no exact match or localname/namespace wildcard is available, fallback
		// to just matching on type alone.
		m: []mux.Option{
			mux.IQ(stanza.ResultIQ, xml.Name{Local: "test", Space: exampleNS}, failHandler{}),
			mux.IQ(stanza.ErrorIQ, xml.Name{}, passHandler{}),
		},
		x:   `<iq type="error" xmlns="jabber:client"><test xmlns="com.example"/></iq>`,
		err: errPassTest,
	},
	13: {
		// Test nop non-stanza handler.
		x: `<nop/>`,
	},
	14: {
		// Test nop message handler.
		m: []mux.Option{mux.Message(stanza.HeadlineMessage, xml.Name{}, failHandler{})},
		x: `<message xml:lang="de" type="chat" xmlns="jabber:server"/>`,
	},
	15: {
		// Test nop presence handler.
		m: []mux.Option{mux.Presence(stanza.SubscribedPresence, xml.Name{}, failHandler{})},
		x: `<presence to="romeo@example.net" from="mercutio@example.net" xmlns="jabber:server"/>`,
	},
	16: {
		// Expect nil IQ handler to panic.
		m:           []mux.Option{mux.IQ(stanza.GetIQ, xml.Name{}, nil)},
		expectPanic: true,
	},
	17: {
		// Expect nil message handler to panic.
		m:           []mux.Option{mux.Message(stanza.ChatMessage, xml.Name{}, nil)},
		expectPanic: true,
	},
	18: {
		// Expect nil presence handler to panic.
		m:           []mux.Option{mux.Presence(stanza.ProbePresence, xml.Name{}, nil)},
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
			mux.Message(stanza.ChatMessage, xml.Name{}, failHandler{}),
			mux.Message(stanza.ChatMessage, xml.Name{}, failHandler{}),
		},
		expectPanic: true,
	},
	22: {
		// Expect duplicate presence handler to panic.
		m: []mux.Option{
			mux.Presence(stanza.ProbePresence, xml.Name{}, failHandler{}),
			mux.Presence(stanza.ProbePresence, xml.Name{}, failHandler{}),
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
	24: {
		// Expect {}message registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "message"}, failHandler{}),
		},
		expectPanic: true,
	},
	25: {
		// Expect {jabber:server}message registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSServer, Local: "message"}, failHandler{}),
		},
		expectPanic: true,
	},
	26: {
		// Expect {jabber:server}message registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSClient, Local: "message"}, failHandler{}),
		},
		expectPanic: true,
	},
	27: {
		// Expect {}presence registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "presence"}, failHandler{}),
		},
		expectPanic: true,
	},
	28: {
		// Expect {jabber:server}presence registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSServer, Local: "presence"}, failHandler{}),
		},
		expectPanic: true,
	},
	29: {
		// Expect {jabber:server}presence registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSClient, Local: "presence"}, failHandler{}),
		},
		expectPanic: true,
	},
	30: {
		// Expect {}iq registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Local: "iq"}, failHandler{}),
		},
		expectPanic: true,
	},
	31: {
		// Expect {jabber:server}iq registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSServer, Local: "iq"}, failHandler{}),
		},
		expectPanic: true,
	},
	32: {
		// Expect {jabber:server}iq registration with Handle to panic
		m: []mux.Option{
			mux.Handle(xml.Name{Space: stanza.NSClient, Local: "iq"}, failHandler{}),
		},
		expectPanic: true,
	},
	33: {
		// If no exact match is available, fallback to the namespace wildcard
		// handler.
		m: []mux.Option{
			mux.Message(stanza.ChatMessage, xml.Name{Local: "test", Space: ""}, passHandler{}),
			mux.Message(stanza.ChatMessage, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<message type="chat" xmlns="jabber:client"><test xmlns="com.example"/></message>`,
		err: errPassTest,
	},
	34: {
		// If no exact match or namespace handler is available, fallback local name
		// handler.
		m: []mux.Option{
			mux.Message(stanza.ChatMessage, xml.Name{Local: "", Space: exampleNS}, passHandler{}),
			mux.Message(stanza.NormalMessage, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<message type="chat" xmlns="jabber:client"><test xmlns="com.example"/></message>`,
		err: errPassTest,
	},
	35: {
		// If no exact match or localname/namespace wildcard is available, fallback
		// to just matching on type alone.
		m: []mux.Option{
			mux.Message(stanza.NormalMessage, xml.Name{Local: "test", Space: exampleNS}, failHandler{}),
			mux.Message(stanza.ChatMessage, xml.Name{}, passHandler{}),
		},
		x:   `<message type="chat" xmlns="jabber:client"><test xmlns="com.example"/></message>`,
		err: errPassTest,
	},
	36: {
		// If no exact match is available, fallback to the namespace wildcard
		// handler.
		m: []mux.Option{
			mux.Presence(stanza.SubscribePresence, xml.Name{Local: "test", Space: ""}, passHandler{}),
			mux.Presence(stanza.SubscribePresence, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<presence type="subscribe" xmlns="jabber:client"><test xmlns="com.example"/></presence>`,
		err: errPassTest,
	},
	37: {
		// If no exact match or namespace handler is available, fallback local name
		// handler.
		m: []mux.Option{
			mux.Presence(stanza.SubscribePresence, xml.Name{Local: "", Space: exampleNS}, passHandler{}),
			mux.Presence(stanza.SubscribedPresence, xml.Name{Local: "", Space: exampleNS}, failHandler{}),
		},
		x:   `<presence type="subscribe" xmlns="jabber:client"><test xmlns="com.example"/></presence>`,
		err: errPassTest,
	},
	38: {
		// If no exact match or localname/namespace wildcard is available, fallback
		// to just matching on type alone.
		m: []mux.Option{
			mux.Presence(stanza.SubscribedPresence, xml.Name{Local: "test", Space: exampleNS}, failHandler{}),
			mux.Presence(stanza.SubscribePresence, xml.Name{}, passHandler{}),
		},
		x:   `<presence type="subscribe" xmlns="jabber:client"><test xmlns="com.example"/></presence>`,
		err: errPassTest,
	},
	39: {
		// Test that multiple handlers can run correctly for presence.
		m: []mux.Option{
			mux.Presence(stanza.SubscribePresence, xml.Name{Local: "test", Space: exampleNS}, multiHandler{}),
			mux.Presence(stanza.SubscribePresence, xml.Name{Local: "example", Space: exampleNS}, multiHandler{}),
		},
		x:   `<presence type="subscribe" xmlns="jabber:client"><example xmlns="com.example">example</example><test xmlns="com.example">test</test></presence>`,
		err: errors.New("mux_test: PASSED, mux_test: PASSED"),
	},
	40: {
		// Test that multiple handlers can run correctly for messages.
		m: []mux.Option{
			mux.Message(stanza.NormalMessage, xml.Name{Local: "test", Space: exampleNS}, multiHandler{}),
			mux.Message(stanza.NormalMessage, xml.Name{Local: "example", Space: exampleNS}, multiHandler{}),
		},
		x:        `<message type="normal" xmlns="jabber:server"><test xmlns="com.example">test</test><example xmlns="com.example">example</example></message>`,
		err:      errors.New("mux_test: PASSED, mux_test: PASSED"),
		stanzaNS: stanza.NSServer,
	},
	41: {
		m: []mux.Option{
			mux.Message(stanza.ChatMessage, xml.Name{}, decodeHandler{}),
		},
		x:   `<message xmlns='jabber:client' type='chat'/>`,
		err: errPassTest,
	},
	42: {
		// An empty IQ of type "get" is illegal and should result in an error.
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{}),
		},
		x:   `<iq xml:lang="en-us" type="get" xmlns="jabber:client"></iq>`,
		err: io.EOF,
	},
	43: {
		// An empty IQ of type "set" is illegal and should result in an error.
		m: []mux.Option{
			mux.IQ(stanza.SetIQ, xml.Name{}, failHandler{}),
		},
		x:   `<iq xml:lang="en-us" type="set" xmlns="jabber:client"></iq>`,
		err: io.EOF,
	},
	44: {
		// An empty IQ of type "result" is fine.
		m: []mux.Option{
			mux.IQ(stanza.ResultIQ, xml.Name{}, passHandler{}),
		},
		x:   `<iq xml:lang="en-us" type="result" xmlns="jabber:client"></iq>`,
		err: errPassTest,
	},
	45: {
		// IQ matches should skip char data (in case the other end of the connection
		// is sending formatted XML)
		m: []mux.Option{
			mux.IQ(stanza.GetIQ, xml.Name{Local: "a"}, passHandler{}),
		},
		x:   `<iq xml:lang="en-us" type="get" xmlns="jabber:client">  <a/></iq>`,
		err: errPassTest,
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
					r := recover()
					if r == nil {
						t.Error("Expected panic")
					}
				}()
			}
			stanzaNS := tc.stanzaNS
			if stanzaNS == "" {
				stanzaNS = stanza.NSClient
			}
			m := mux.New(stanzaNS, tc.m...)
			d := xml.NewDecoder(strings.NewReader(tc.x))
			tok, _ := d.Token()
			start, ok := tok.(xml.StartElement)
			if !ok {
				t.Fatalf("did not get start element, got token %v of type %[1]T", tok)
			}

			err := m.HandleXMPP(nopEncoder{TokenReader: d}, &start)
			switch {
			case tc.err == nil && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.err != nil && err == nil:
				t.Fatalf("got nil error but expected %v", tc.err)
			case tc.err == nil && err == nil:
				// All good
			case tc.err.Error() != err.Error():
				t.Fatalf("unexpected error: want=%v, got=%v", tc.err.Error(), err.Error())
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

var fallbackTestCases = [...]struct {
	in  string
	out string
}{
	0: {
		in:  `<iq xmlns="jabber:client" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`,
		out: `<iq xmlns="jabber:client" type="error" to="juliet@example.com" from="romeo@example.com" id="123"><error type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable></error></iq>`,
	},
	1: {
		in:  `<iq xmlns="jabber:client" type="get" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`,
		out: `<iq xmlns="jabber:client" type="error" to="juliet@example.com" from="romeo@example.com" id="123"><error type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable></error></iq>`,
	},
	2: {
		in:  `<iq xmlns="jabber:client" type="set" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`,
		out: `<iq xmlns="jabber:client" type="error" to="juliet@example.com" from="romeo@example.com" id="123"><error type="cancel"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable></error></iq>`,
	},
	3: {
		in: `<iq xmlns="jabber:client" type="error" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`,
	},
	4: {
		in: `<iq xmlns="jabber:client" type="result" to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`,
	},
}

func TestFallback(t *testing.T) {
	for i, tc := range fallbackTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: strings.NewReader(tc.in),
				Writer: buf,
			}
			s := xmpptest.NewClientSession(0, rw)

			r := s.TokenReader()
			defer r.Close()
			tok, err := r.Token()
			if err != nil {
				t.Fatalf("Bad start token read: `%v'", err)
			}
			start := tok.(xml.StartElement)
			w := s.TokenWriter()
			defer w.Close()
			err = mux.New(stanza.NSClient).HandleXMPP(testEncoder{
				TokenReader: r,
				TokenWriter: w,
			}, &start)
			if err != nil {
				t.Errorf("Unexpected error: `%v'", err)
			}
			if err := w.Flush(); err != nil {
				t.Errorf("Unexpected error flushing token writer: %q", err)
			}

			if out := buf.String(); out != tc.out {
				t.Errorf("Bad output:\nwant=`%v'\n got=`%v'", tc.out, out)
			}
		})
	}
}

func TestLazyServeMuxMapInitialization(t *testing.T) {
	m := &mux.ServeMux{}

	// This will panic if the map isn't initialized lazily.
	mux.Handle(xml.Name{}, failHandler{})(m)
	mux.IQ(stanza.GetIQ, xml.Name{}, failHandler{})(m)
	mux.Message(stanza.NormalMessage, xml.Name{}, failHandler{})(m)
	mux.Presence(stanza.SubscribePresence, xml.Name{}, failHandler{})(m)
}

const (
	testFeature         = "urn:example"
	iqTestFeature       = "urn:example:iq"
	msgTestFeature      = "urn:example:message"
	presenceTestFeature = "urn:example:presence"
	otherTestFeature    = "urn:example:other"
)

type handleFeature struct{}

func (handleFeature) HandleXMPP(xmlstream.TokenReadEncoder, *xml.StartElement) error {
	panic("should not be called")
}

func (handleFeature) ForFeatures(node string, f func(info.Feature) error) error {
	return f(info.Feature{
		Var: testFeature,
	})
}

func (handleFeature) ForItems(node string, f func(items.Item) error) error {
	return f(items.Item{
		Name: testFeature,
	})
}

type iqFeature struct{}

func (iqFeature) HandleIQ(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {
	panic("should not be called")
}

func (iqFeature) ForFeatures(node string, f func(info.Feature) error) error {
	return f(info.Feature{
		Var: iqTestFeature,
	})
}

func (iqFeature) ForItems(node string, f func(items.Item) error) error {
	return f(items.Item{
		Name: iqTestFeature,
	})
}

type messageFeature struct{}

func (messageFeature) HandleMessage(stanza.Message, xmlstream.TokenReadEncoder) error {
	panic("should not be called")
}

func (messageFeature) ForFeatures(node string, f func(info.Feature) error) error {
	return f(info.Feature{
		Var: msgTestFeature,
	})
}

func (messageFeature) ForItems(node string, f func(items.Item) error) error {
	return f(items.Item{
		Name: msgTestFeature,
	})
}

type presenceFeature struct{}

func (presenceFeature) HandlePresence(stanza.Presence, xmlstream.TokenReadEncoder) error {
	panic("should not be called")
}

func (presenceFeature) ForFeatures(node string, f func(info.Feature) error) error {
	return f(info.Feature{
		Var: presenceTestFeature,
	})
}

func (presenceFeature) ForItems(node string, f func(items.Item) error) error {
	return f(items.Item{
		Name: presenceTestFeature,
	})
}

type otherFeature struct{}

func (otherFeature) ForFeatures(node string, f func(info.Feature) error) error {
	return f(info.Feature{
		Var: otherTestFeature,
	})
}
func (otherFeature) ForIdentities(node string, f func(info.Identity) error) error {
	return f(info.Identity{
		Name: otherTestFeature,
	})
}

func TestFeatures(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Handle(xml.Name{}, handleFeature{}),
		mux.IQ("", xml.Name{}, iqFeature{}),
		mux.Message("", xml.Name{}, messageFeature{}),
		mux.Presence("", xml.Name{}, presenceFeature{}),
		mux.Feature(otherFeature{}),
		mux.Ident(otherFeature{}),
	)
	var (
		foundHandler  bool
		foundIQ       bool
		foundPresence bool
		foundMsg      bool
		foundOther    bool
	)
	err := m.ForFeatures("", func(i info.Feature) error {
		switch i.Var {
		case testFeature:
			foundHandler = true
		case iqTestFeature:
			foundIQ = true
		case msgTestFeature:
			foundMsg = true
		case presenceTestFeature:
			foundPresence = true
		case otherTestFeature:
			foundOther = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error while iterating over features: %v", err)
	}
	if !foundHandler {
		t.Errorf("features iter did not find plain handler feature")
	}
	if !foundIQ {
		t.Errorf("features iter did not find IQ feature")
	}
	if !foundMsg {
		t.Errorf("features iter did not find message feature")
	}
	if !foundPresence {
		t.Errorf("features iter did not find presence feature")
	}
	if !foundOther {
		t.Errorf("features iter did not find other test feature")
	}
	var foundOtherIdent bool
	err = m.ForIdentities("", func(i info.Identity) error {
		if i.Name == otherTestFeature {
			foundOtherIdent = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error while iterating over identities: %v", err)
	}
	if !foundOtherIdent {
		t.Errorf("ident iter did not find other test identity")
	}
}

func TestItems(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Handle(xml.Name{}, handleFeature{}),
		mux.IQ("", xml.Name{}, iqFeature{}),
		mux.Message("", xml.Name{}, messageFeature{}),
		mux.Presence("", xml.Name{}, presenceFeature{}),
	)
	var (
		foundHandler  bool
		foundIQ       bool
		foundPresence bool
		foundMsg      bool
	)
	err := m.ForItems("", func(i items.Item) error {
		switch i.Name {
		case testFeature:
			foundHandler = true
		case iqTestFeature:
			foundIQ = true
		case msgTestFeature:
			foundMsg = true
		case presenceTestFeature:
			foundPresence = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error while iterating over features: %v", err)
	}
	if !foundHandler {
		t.Errorf("items iter did not find plain handler item")
	}
	if !foundIQ {
		t.Errorf("items iter did not find IQ item")
	}
	if !foundMsg {
		t.Errorf("items iter did not find message item")
	}
	if !foundPresence {
		t.Errorf("items iter did not find presence item")
	}
}

func TestFeaturesHandlerErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Handle(xml.Name{}, handleFeature{}),
	)
	err := m.ForFeatures("", func(i info.Feature) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}

func TestFeaturesHandleIQErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.IQ("", xml.Name{}, iqFeature{}),
	)
	err := m.ForFeatures("", func(i info.Feature) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
func TestFeaturesHandleMsgErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Message("", xml.Name{}, messageFeature{}),
	)
	err := m.ForFeatures("", func(i info.Feature) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
func TestFeaturesHandlePresenceErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Presence("", xml.Name{}, presenceFeature{}),
	)
	err := m.ForFeatures("", func(i info.Feature) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}

func TestItemsHandlerErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Handle(xml.Name{}, handleFeature{}),
	)
	err := m.ForItems("", func(i items.Item) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
func TestItemsHandleIQErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.IQ("", xml.Name{}, iqFeature{}),
	)
	err := m.ForItems("", func(i items.Item) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
func TestItemsHandleMsgErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Message("", xml.Name{}, messageFeature{}),
	)
	err := m.ForItems("", func(i items.Item) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
func TestItemsHandlePresenceErr(t *testing.T) {
	m := mux.New(
		stanza.NSClient,
		mux.Presence("", xml.Name{}, presenceFeature{}),
	)
	err := m.ForItems("", func(i items.Item) error {
		return io.EOF
	})
	if err != io.EOF {
		t.Fatalf("wrong error: want=%v, got=%v", io.EOF, err)
	}
}
