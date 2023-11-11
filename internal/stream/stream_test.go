// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/internal/decl"
	intstream "mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

var expectTestCases = [...]struct {
	XML  string
	Err  bool
	Recv bool
	WS   bool
}{
	0: {Err: true},
	1: {
		XML: "<?xml version='1.0'?>\n\t <stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0' id='123'>",
	},
	2: {
		XML: "<?xml version='1.0'?><stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0' id='123'>",
	},
	3: {
		XML: "<stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0' id='123'>",
	},
	4: {
		XML: `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" to="example.com" version="1.0" id="123" />`,
		WS:  true,
	},
	5: {
		XML:  `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" to="example.com" version="1.0" />`,
		WS:   true,
		Recv: true,
	},
	6: {
		XML: "<foo/>",
		Err: true,
	},
	7: {
		XML: "<foo/>",
		WS:  true,
		Err: true,
	},
	8: {
		XML: "<?xml version='1.0'?>",
		Err: true,
	},
	9: {
		// TODO: is this actually legal? I don't see why it wouldn't be, but I have
		// a vague recollection that the first byte in an XML stream was always '<'.
		// This test may need to change if we find out this is wrong.
		XML: "\n\t <stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0' id='123'>",
	},
	10: {
		XML: `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" to="example.com" version="1.0" id="123">`,
		WS:  true,
		Err: true,
	},
	11: {
		XML: "<stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='0.0' id='123'>",
		Err: true,
	},
	12: {
		XML: "<stream:stream xmlns='jabber:foo' xmlns:stream='http://etherx.jabber.org/streams' version='1.0' id='123'>",
		Err: true,
	},
	13: {
		XML: "<stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0'>",
		Err: true,
	},
	14: {
		XML: "<stream:stream xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='0' id='123'>",
		Err: true,
	},
}

func TestExpect(t *testing.T) {
	for i, tc := range expectTestCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := xml.NewDecoder(strings.NewReader(tc.XML))
			info := &stream.Info{}
			err := intstream.Expect(context.Background(), info, d, tc.Recv, tc.WS)
			switch {
			case err != nil && !tc.Err:
				t.Errorf("Did not expect error but got %v", err)
			case err == nil && tc.Err:
				t.Error("Expected error but did not get one")
			}
		})
	}
}

func TestSendNewS2S(t *testing.T) {
	for i, tc := range []struct {
		s2s    bool
		id     bool
		output string
		err    error
	}{
		{true, true, ` id='abc' `, nil},
		{false, true, ` id='abc' `, nil},
		{true, true, ` xmlns='jabber:server' `, nil},
		{false, false, ` xmlns='jabber:client' `, nil},
	} {
		name := fmt.Sprintf("%d", i)
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			ids := ""
			if tc.id {
				ids = "abc"
			}
			out := stream.Info{}
			if tc.s2s {
				out.XMLNS = stanza.NSServer
			} else {
				out.XMLNS = stanza.NSClient
			}
			err := intstream.Send(&b, &out, false, stream.Version{Major: 1, Minor: 0}, "und", "example.net", "test@example.net", ids)

			str := b.String()
			if !strings.HasPrefix(str, decl.XMLHeader) {
				t.Errorf("Expected string to start with XML header but got: %s", str)
			}
			str = strings.TrimPrefix(str, decl.XMLHeader)

			switch {
			case err != tc.err:
				t.Errorf("Error did not match, excepted `%s` but got `%s`.", tc.err, err)
			case !strings.Contains(str, tc.output):
				t.Errorf("Expected string to contain `%s` but got: %s", tc.output, str)
			case !strings.HasPrefix(str, `<stream:stream `):
				t.Errorf("Expected string to start with `<stream:stream ` but got: %s", str)
			case !strings.Contains(str, ` to='example.net' `):
				t.Errorf("Expected string to contain ` to='example.net' ` but got: %s", str)
			case !strings.Contains(str, ` from='test@example.net' `):
				t.Errorf("Expected string to contain ` from='test@example.net' ` but got: %s", str)
			case !strings.Contains(str, ` version='1.0' `):
				t.Errorf("Expected string to contain ` version='1.0' ` but got: %s", str)
			case !strings.Contains(str, ` xml:lang='und'`):
				t.Errorf("Expected string to contain ` xml:lang='und'` but got: %s", str)
			case !strings.Contains(str, ` xmlns:stream='http://etherx.jabber.org/streams'`):
				t.Errorf("Expected string to contain xmlns:stream=… but got: %s", str)
			}
		})
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

type nopReader struct{}

func (nopReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

func TestSendNewS2SReturnsWriteErr(t *testing.T) {
	out := stream.Info{}
	err := intstream.Send(struct {
		io.Reader
		io.Writer
	}{
		nopReader{},
		errWriter{},
	}, &out, false, stream.Version{Major: 1, Minor: 0}, "und", "example.net", "test@example.net", "abc")
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Expected errWriterErr (%s) but got `%s`", io.ErrUnexpectedEOF, err)
	}
}
