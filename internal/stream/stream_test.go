// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/internal/decl"
	intstream "mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

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

var expectTestCases = [...]struct {
	XML            string
	Err            error
	Recv           bool
	WS             bool
	SkipFinalToken bool
	Pop            int
}{
	0: {
		Err:            io.EOF,
		SkipFinalToken: true,
	},
	1: {
		XML: xml.Header[:len(xml.Header)-1] + `<stream><fin/>`,
		Err: stream.InvalidNamespace,
	},
	2: {
		XML: xml.Header[:len(xml.Header)-1] + `<open><fin/>`,
		WS:  true,
		Err: stream.InvalidNamespace,
	},
	3: {
		XML: xml.Header[:len(xml.Header)-1] + xml.Header[:len(xml.Header)-1] + `<fin/>`,
		WS:  true,
		Err: stream.RestrictedXML,
	},
	4: {
		XML: `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing"
		            to="example.com"
								version="1.0" /><fin/>`,
		Recv: true,
		WS:   true,
	},
	5: {
		XML: `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing"
		            to="example.com"
								version="1.0" /><fin/>`,
		WS:  true,
		Err: stream.BadFormat,
	},
	6: {
		XML: `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing"
		            to="example.com"
								version="2.0" /><fin/>`,
		Recv: true,
		WS:   true,
		Err:  stream.UnsupportedVersion,
	},
	7: {
		XML: `<stream:stream
            from='juliet@im.example.com'
            to='im.example.com'
            version='1.0'
            xml:lang='en'
						id='1234'
            xmlns='wrong'
            xmlns:stream='http://etherx.jabber.org/streams'><fin/>`,
		Err: stream.InvalidNamespace,
	},
	8: {
		XML: `<stream></stream><fin/>`,
		Err: stream.NotWellFormed,
		Pop: 1,
	},
	9: {
		XML: `<!-- test --><fin/>`,
		Err: stream.RestrictedXML,
	},
}

func TestExpect(t *testing.T) {
	for i, tc := range expectTestCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := xml.NewDecoder(strings.NewReader(tc.XML))
			for ; tc.Pop > 0; tc.Pop-- {
				_, err := d.Token()
				if err != nil {
					t.Fatalf("error while poping start tokens: %v", err)
				}
			}
			info := &stream.Info{}
			err := intstream.Expect(context.Background(), info, d, tc.Recv, tc.WS)
			if (tc.Err != nil && !errors.Is(err, tc.Err)) || (tc.Err == nil && err != nil) {
				t.Fatalf("wrong error: want=%v, got=%v", tc.Err, err)
			}
			if !tc.SkipFinalToken {
				tok, err := d.Token()
				if err != nil {
					t.Fatalf("error reading expected final token: %v", err)
				}
				start, ok := tok.(xml.StartElement)
				if !ok || start.Name.Local != "fin" {
					t.Fatalf("unexpected final token: %T(%[1]v)", tok)
				}
			}
		})
	}
}
