// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/jid"
)

func TestSendNewS2S(t *testing.T) {
	config := NewClientConfig(jid.MustParse("test@example.net"))
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
			config.S2S = tc.s2s
			err := sendNewStream(&b, config, ids)

			switch {
			case err != tc.err:
				t.Errorf("Error did not match, excepted `%s` but got `%s`.", tc.err, err)
			case !strings.Contains(b.String(), tc.output):
				t.Errorf("Expected string to contain `%s` but got: %s", tc.output, b.String())
			case !strings.HasPrefix(b.String(), `<stream:stream `):
				t.Errorf("Expected string to start with `<stream:stream ` but got: %s", b.String())
			case !strings.Contains(b.String(), ` to='example.net' `):
				t.Errorf("Expected string to contain ` to='example.net' ` but got: %s", b.String())
			case !strings.Contains(b.String(), ` from='test@example.net' `):
				t.Errorf("Expected string to contain ` from='test@example.net' ` but got: %s", b.String())
			case !strings.Contains(b.String(), ` version='1.0' `):
				t.Errorf("Expected string to contain ` version='1.0' ` but got: %s", b.String())
			case !strings.Contains(b.String(), ` xml:lang='und' `):
				t.Errorf("Expected string to contain ` xml:lang='und' ` but got: %s", b.String())
			case !strings.HasSuffix(b.String(), ` xmlns:stream='http://etherx.jabber.org/streams'>`):
				t.Errorf("Expected string to end with xmlns:stream=… but got: %s", b.String())
			}
		})
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestSendNewS2SReturnsWriteErr(t *testing.T) {
	config := NewClientConfig(jid.MustParse("test@example.net"))
	if err := sendNewStream(&errWriter{}, config, "abc"); err != io.ErrUnexpectedEOF {
		t.Errorf("Expected errWriterErr (%s) but got `%s`", io.ErrUnexpectedEOF, err)
	}
}

type rwNopCloser struct {
	io.ReadWriter
}

func (rwNopCloser) Close() error { return nil }

func TestSendNewS2SClearsStreamRestartBit(t *testing.T) {
	var b bytes.Buffer
	rwc := rwNopCloser{&b}
	config := NewClientConfig(jid.MustParse("test@example.net"))
	conn := &Conn{
		state: StreamRestartRequired | Bind,
		rwc:   rwc,
	}
	err := sendNewStream(conn, config, "abc")
	if err != nil {
		t.Error(err)
	}
	if conn.state&StreamRestartRequired != 0 {
		t.Error("Expected sending a new stream to clear the StreamRestartRequired bit.")
	}
}
