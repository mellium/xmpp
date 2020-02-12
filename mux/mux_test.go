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
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
)

var passTest = errors.New("mux_test: PASSED")

var passHandler xmpp.HandlerFunc = func(xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return passTest
}

var failHandler xmpp.HandlerFunc = func(xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return errors.New("mux_test: FAILED")
}

var testCases = [...]struct {
	m *mux.ServeMux
	p xml.Name
}{
	0: {
		m: mux.New(mux.IQ(passHandler), mux.Presence(failHandler)),
		p: xml.Name{Local: "iq", Space: ns.Client},
	},
	1: {
		m: mux.New(mux.IQFunc(passHandler), mux.Presence(failHandler)),
		p: xml.Name{Local: "iq", Space: ns.Server},
	},
	2: {
		m: mux.New(mux.IQ(failHandler), mux.Message(passHandler)),
		p: xml.Name{Local: "message", Space: ns.Client},
	},
	3: {
		m: mux.New(mux.IQ(failHandler), mux.MessageFunc(passHandler)),
		p: xml.Name{Local: "message", Space: ns.Server},
	},
	4: {
		m: mux.New(mux.Message(failHandler), mux.IQ(failHandler), mux.Presence(passHandler)),
		p: xml.Name{Local: "presence", Space: ns.Client},
	},
	5: {
		m: mux.New(mux.Message(failHandler), mux.IQ(failHandler), mux.PresenceFunc(passHandler)),
		p: xml.Name{Local: "presence", Space: ns.Server},
	},
	6: {
		m: mux.New(mux.IQ(passHandler)),
		p: xml.Name{Local: "iq", Space: ns.Server},
	},
	7: {
		m: mux.New(mux.Handle(xml.Name{Local: "test"}, passHandler)),
		p: xml.Name{Local: "test", Space: "summertime"},
	},
	8: {
		m: mux.New(mux.HandleFunc(xml.Name{Space: "summertime"}, passHandler)),
		p: xml.Name{Local: "test", Space: "summertime"},
	},
}

func TestMux(t *testing.T) {
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := tc.m.HandleXMPP(nil, &xml.StartElement{Name: tc.p})
			if err != passTest {
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
		Reader: strings.NewReader(`<iq to="romeo@example.com" from="juliet@example.com" id="123"><test/></iq>`),
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

func TestNilHandlerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic when trying to register a nil handler")
		}
	}()
	mux.New(mux.IQ(nil))
}

func TestIdenticalHandlerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic when trying to register a duplicate handler")
		}
	}()
	mux.New(mux.IQ(failHandler), mux.IQ(failHandler))
}

func TestLazyServeMuxMapInitialization(t *testing.T) {
	m := &mux.ServeMux{}

	// This will panic if the map isn't initialized lazily.
	mux.IQ(failHandler)(m)
}
