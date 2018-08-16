// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"crypto/tls"
	"errors"
	"io"
	"strconv"
	"testing"
)

var errClose = errors.New("test close error")

type errCloser struct {
	io.ReadWriter
}

func (errCloser) Close() error {
	return errClose
}

var connTestCases = [...]struct {
	rw  io.ReadWriter
	err error
}{
	0: {rw: struct{ io.ReadWriter }{}},
	1: {rw: &tls.Conn{}},
	2: {rw: errCloser{}, err: errClose},
}

func TestConn(t *testing.T) {
	for i, tc := range connTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			conn := newConn(tc.rw)

			_, isTLSConn := tc.rw.(*tls.Conn)
			if ok := conn.Secure(); ok != isTLSConn {
				t.Errorf("TLS conn not wrapped properly: want=%t, got=%t", isTLSConn, ok)
			}

			// Don't run closer tests against dummy TLS connections that will panic.
			if !isTLSConn {
				if err := conn.Close(); err != tc.err {
					t.Errorf("Unexpected error closing conn: want=%q, got=%q", tc.err, err)
				}
			}
		})
	}
}
