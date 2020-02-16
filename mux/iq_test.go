// Copyright 2020 The Mellium Contributors.
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
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const exampleNS = "com.example"

var passIQHandler mux.IQHandlerFunc = func(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return passTest
}

var failIQHandler mux.IQHandlerFunc = func(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {
	return errors.New("mux_test: FAILED")
}

var iqTestCases = [...]struct {
	m      *mux.IQMux
	p      xml.Name
	iqType stanza.IQType
}{
	0: {
		// Exact match handler should be selected if available.
		m: mux.NewIQMux(
			mux.HandleIQ(stanza.GetIQ, xml.Name{Local: "a", Space: exampleNS}, failIQHandler),
			mux.HandleIQ(stanza.GetIQ, xml.Name{Local: "test", Space: "b"}, failIQHandler),
			mux.HandleIQFunc(stanza.GetIQ, xml.Name{Local: "test", Space: exampleNS}, passIQHandler),
		),
		p: xml.Name{Local: "test", Space: exampleNS},
	},
	1: {
		// If no exact match is available, fallback to the namespace wildcard
		// handler.
		m: mux.NewIQMux(
			mux.GetIQFunc(xml.Name{Local: "test", Space: ""}, passIQHandler),
			mux.HandleIQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, failIQHandler),
		),
		p: xml.Name{Local: "test", Space: exampleNS},
	},
	2: {
		// If no exact match or namespace handler is available, fallback local name
		// handler.
		m: mux.NewIQMux(
			mux.HandleIQ(stanza.GetIQ, xml.Name{Local: "", Space: exampleNS}, passIQHandler),
		),
		p: xml.Name{Local: "test", Space: exampleNS},
	},
	3: {
		// If no exact match or localname/namespace wildcard is available, fallback
		// to just matching on type alone.
		m: mux.NewIQMux(
			mux.ResultIQFunc(xml.Name{Local: "test", Space: exampleNS}, failIQHandler),
			mux.ErrorIQFunc(passIQHandler),
		),
		p:      xml.Name{Local: "test", Space: exampleNS},
		iqType: stanza.ErrorIQ,
	},
	4: {
		// IQs must be routed correctly by type.
		m: mux.NewIQMux(
			mux.GetIQ(xml.Name{Local: "test", Space: exampleNS}, failIQHandler),
			mux.SetIQFunc(xml.Name{Local: "test", Space: exampleNS}, failIQHandler),
			mux.ResultIQ(xml.Name{Local: "test", Space: exampleNS}, passIQHandler),
			mux.ErrorIQ(passIQHandler),
		),
		p:      xml.Name{Local: "test", Space: exampleNS},
		iqType: stanza.ResultIQ,
	},
}

func TestIQMux(t *testing.T) {
	for i, tc := range iqTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			iqType := stanza.GetIQ
			if tc.iqType != "" {
				iqType = tc.iqType
			}
			err := tc.m.HandleXMPP(
				testReadEncoder{xmlstream.Wrap(nil, xml.StartElement{Name: tc.p})},
				&xml.StartElement{Name: xml.Name{Local: "iq"}, Attr: []xml.Attr{{Name: xml.Name{Local: "type"}, Value: string(iqType)}}},
			)
			if err != passTest {
				t.Fatalf("unexpected error: `%v'", err)
			}
		})
	}
}

type testReadEncoder struct {
	xml.TokenReader
}

func (testReadEncoder) EncodeToken(t xml.Token) error { return nil }

func (testReadEncoder) EncodeElement(interface{}, xml.StartElement) error {
	panic("unexpected EncodeElement")
}

func (testReadEncoder) Encode(interface{}) error { panic("unexpected Encode") }

func TestIQFallback(t *testing.T) {
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
	err = mux.NewIQMux().HandleXMPP(testEncoder{
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

func TestNilIQHandlerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic when trying to register a nil IQ handler")
		}
	}()
	mux.NewIQMux(mux.GetIQ(xml.Name{}, nil))
}

func TestIdenticalIQHandlerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic when trying to register a duplicate IQ handler")
		}
	}()
	mux.NewIQMux(
		mux.GetIQ(xml.Name{Space: "space", Local: "local"}, failIQHandler),
		mux.GetIQ(xml.Name{Space: "space", Local: "local"}, failIQHandler),
	)
}

func TestLazyIQMuxMapInitialization(t *testing.T) {
	m := &mux.IQMux{}

	// This will panic if the map isn't initialized lazily.
	mux.GetIQ(xml.Name{}, failIQHandler)(m)
}
