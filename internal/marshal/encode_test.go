// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package marshal_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/stanza"
)

type testWriteFlusher struct {
	flushes int
}

func (wf *testWriteFlusher) EncodeToken(t xml.Token) error {
	return nil
}

func (wf *testWriteFlusher) Flush() error {
	wf.flushes++
	return nil
}

type noopTokenWriter struct{}

func (noopTokenWriter) EncodeToken(xml.Token) error { return nil }

type errTokenWriter struct{}

func (errTokenWriter) EncodeToken(xml.Token) error { return bytes.ErrTooLarge }

var errRead = fmt.Errorf("sentinal error")

type errTokenReader struct{}

func (errTokenReader) Token() (xml.Token, error) {
	return nil, errRead
}

type errWriterTo struct{}

func (errWriterTo) WriteXML(xmlstream.TokenWriter) (int, error) {
	return 23, errRead
}

type errMarshaler struct{}

func (errMarshaler) TokenReader() xml.TokenReader { return errTokenReader{} }

var simpleIn = struct {
	XMLName xml.Name `xml:"space local"`
}{}

var encodeTestCases = [...]struct {
	w       xmlstream.TokenWriter
	v       interface{}
	err     error
	errType error
}{
	0: {
		v:       struct{}{},
		errType: &xml.UnsupportedTypeError{},
	},
	1: {
		w: noopTokenWriter{},
		v: simpleIn,
	},
	2: {
		w:   errTokenWriter{},
		v:   simpleIn,
		err: bytes.ErrTooLarge,
	},
	3: {
		// If the type is a marshaler it should be used (not encoded itself).
		v:   errMarshaler{},
		err: errRead,
	},
	4: {
		// If the type is a TokenReader it should be used (not encoded itself).
		v:   errTokenReader{},
		err: errRead,
	},
	5: {
		// If the type is a WriterTo call its WriteXML method instead of creating a
		// decoder.
		v:   errWriterTo{},
		err: errRead,
	},
}

func TestEncode(t *testing.T) {
	for i, tc := range encodeTestCases {
		t.Run(fmt.Sprintf("%d/EncodeXML", i), func(t *testing.T) {
			err := marshal.EncodeXML(tc.w, tc.v)
			switch {
			case tc.err != nil && err != tc.err:
				t.Errorf("unexpected error: want=%v, got=%v", tc.err, err)
			case tc.errType != nil && reflect.TypeOf(err) != reflect.TypeOf(tc.errType):
				t.Errorf("error of wrong type: want=%T, got=%T", tc.errType, err)
			}
		})
		t.Run(fmt.Sprintf("%d/EncodeXMLElement", i), func(t *testing.T) {
			err := marshal.EncodeXMLElement(tc.w, tc.v, xml.StartElement{
				Name: xml.Name{Space: "space", Local: "local"},
			})
			switch {
			case tc.err != nil && err != tc.err:
				t.Errorf("unexpected error: want=%v, got=%v", tc.err, err)
			case tc.errType != nil && reflect.TypeOf(err) != reflect.TypeOf(tc.errType):
				t.Errorf("error of wrong type: want=%T, got=%T", tc.errType, err)
			}
		})
	}
}

func TestFlusher(t *testing.T) {
	t.Run("EncodeXML", func(t *testing.T) {
		f := &testWriteFlusher{}
		if err := marshal.EncodeXML(f, 1); err != nil {
			t.Fatal(err)
		}

		if f.flushes != 1 {
			t.Errorf("Expected 1 flush call got %d", f.flushes)
		}
	})
	t.Run("EncodeXMLElement", func(t *testing.T) {
		f := &testWriteFlusher{}
		start := xml.StartElement{Name: xml.Name{Local: "int"}}
		if err := marshal.EncodeXMLElement(f, 1, start); err != nil {
			t.Fatal(err)
		}

		if f.flushes != 1 {
			t.Errorf("Expected 1 flush call got %d", f.flushes)
		}
	})
}

func TestMarshalTokenReader(t *testing.T) {
	r := stanza.IQ{}.Wrap(nil)
	rr, err := marshal.TokenReader(r)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r != rr {
		t.Errorf("got different xml.TokenReader out: want=%v, got=%v", r, rr)
	}
}

func TestTokenReaderMarshaler(t *testing.T) {
	// When passing a marshaler to TokenReader we should use the marshaler instead
	// of encoding the value itself.
	rr, err := marshal.TokenReader(errMarshaler{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = rr.Token()
	if err != errRead {
		t.Errorf("unexpected error from marshaler: %v", err)
	}
}

func TestTokenDecoder(t *testing.T) {
	r := stanza.IQ{}
	_, err := marshal.TokenReader(r)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEncodeXMLNS(t *testing.T) {
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	err := marshal.EncodeXML(e, simpleIn)
	if err != nil {
		t.Errorf("unexpected error encoding: %v", err)
	}
	const expected = `<local xmlns="space"></local>`
	if s := buf.String(); s != expected {
		t.Errorf("wrong output: want=%s, got=%s", expected, s)
	}
}
