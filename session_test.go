// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"

	"mellium.im/xmpp"
)

var errTestNegotiate = errors.New("a test error")

func errNegotiator(ctx context.Context, session *xmpp.Session, data interface{}) (mask xmpp.SessionState, rw io.ReadWriter, cache interface{}, err error) {
	err = errTestNegotiate
	return
}

type negotiateTestCase struct {
	negotiator xmpp.Negotiator
	err        error
	panics     bool
}

var negotiateTests = [...]negotiateTestCase{
	0: {panics: true},
	1: {negotiator: errNegotiator, err: errTestNegotiate},
}

func TestNegotiator(t *testing.T) {
	for i, tc := range negotiateTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case tc.panics && r == nil:
					t.Error("Expected nil negotiator to cause a panic")
				case !tc.panics && r != nil:
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			// TODO: This is just some junk for now. Fix it up when you add more tests
			// that actually need it.
			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: rand.New(rand.NewSource(99)),
				Writer: ioutil.Discard,
			}
			_, err := xmpp.NegotiateSession(context.Background(), nil, nil, rw, tc.negotiator)
			if err != tc.err {
				t.Errorf("Unexpected error: want=%v, got=%v", tc.err, err)
			}
		})
	}
}
