// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
)

func TestClosedInputStream(t *testing.T) {
	for i := 0; i <= math.MaxUint8; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			mask := xmpp.SessionState(i)
			buf := new(bytes.Buffer)
			s := xmpptest.NewSession(mask, buf)

			_, err := s.Token()
			switch {
			case mask&xmpp.InputStreamClosed == xmpp.InputStreamClosed && err != xmpp.ErrInputStreamClosed:
				t.Errorf("Unexpected error: want=`%v', got=`%v'", xmpp.ErrInputStreamClosed, err)
			case mask&xmpp.InputStreamClosed == 0 && err != io.EOF:
				t.Errorf("Unexpected error: `%v'", err)
			}
		})
	}
}

func TestClosedOutputStream(t *testing.T) {
	for i := 0; i <= math.MaxUint8; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			mask := xmpp.SessionState(i)
			buf := new(bytes.Buffer)
			s := xmpptest.NewSession(mask, buf)

			switch err := s.EncodeToken(xml.CharData("chartoken")); {
			case mask&xmpp.OutputStreamClosed == xmpp.OutputStreamClosed && err != xmpp.ErrOutputStreamClosed:
				t.Errorf("Unexpected error: want=`%v', got=`%v'", xmpp.ErrOutputStreamClosed, err)
			case mask&xmpp.OutputStreamClosed == 0 && err != nil:
				t.Errorf("Unexpected error: `%v'", err)
			}
			switch err := s.Flush(); {
			case mask&xmpp.OutputStreamClosed == xmpp.OutputStreamClosed && err != xmpp.ErrOutputStreamClosed:
				t.Errorf("Unexpected error: want=`%v', got=`%v'", xmpp.ErrOutputStreamClosed, err)
			case mask&xmpp.OutputStreamClosed == 0 && err != nil:
				t.Errorf("Unexpected error: `%v'", err)
			}
		})
	}
}

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
			_, err := xmpp.NegotiateSession(context.Background(), jid.JID{}, jid.JID{}, rw, tc.negotiator)
			if err != tc.err {
				t.Errorf("Unexpected error: want=%v, got=%v", tc.err, err)
			}
		})
	}
}
