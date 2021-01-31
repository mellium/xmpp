// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/internal/decl"
	intstream "mellium.im/xmpp/internal/stream"
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
			_, err := intstream.Send(&b, tc.s2s, false, stream.Version{Major: 1, Minor: 0}, "und", "example.net", "test@example.net", ids)

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
	_, err := intstream.Send(struct {
		io.Reader
		io.Writer
	}{
		nopReader{},
		errWriter{},
	}, true, false, stream.Version{Major: 1, Minor: 0}, "und", "example.net", "test@example.net", "abc")
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Expected errWriterErr (%s) but got `%s`", io.ErrUnexpectedEOF, err)
	}
}
