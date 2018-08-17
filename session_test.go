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
	"math"
	"strconv"
	"strings"
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

func TestNilNegotiatorPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic, did not get one")
		}
	}()
	xmpp.NegotiateSession(context.Background(), jid.JID{}, jid.JID{}, nil, nil)
}

var errTestNegotiate = errors.New("a test error")

func errNegotiator(ctx context.Context, session *xmpp.Session, data interface{}) (mask xmpp.SessionState, rw io.ReadWriter, cache interface{}, err error) {
	err = errTestNegotiate
	return
}

type negotiateTestCase struct {
	negotiator xmpp.Negotiator
	in         string
	out        string
	location   jid.JID
	origin     jid.JID
	err        error
}

var negotiateTests = [...]negotiateTestCase{
	0: {negotiator: errNegotiator, err: errTestNegotiate},
	1: {
		negotiator: xmpp.NewNegotiator(xmpp.StreamConfig{
			Features: []xmpp.StreamFeature{xmpp.StartTLS(true, nil)},
		}),
		in:  `<stream:stream id='316732270768047465' version='1.0' xml:lang='en' xmlns:stream='http://etherx.jabber.org/streams' xmlns='jabber:client'><stream:features><other/></stream:features>`,
		out: `<?xml version="1.0" encoding="UTF-8"?><stream:stream to='' from='' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`,
		err: errors.New("XML syntax error on line 1: unexpected EOF"),
	},
	2: {
		negotiator: xmpp.NewNegotiator(xmpp.StreamConfig{}),
		in:         `<stream:stream id='316732270768047465' version='1.0' xml:lang='en' xmlns:stream='http://etherx.jabber.org/streams' xmlns='jabber:client'><stream:features><other/></stream:features>`,
		out:        `<?xml version="1.0" encoding="UTF-8"?><stream:stream to='' from='' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`,
		err:        errors.New("xmpp: features advertised out of order"),
	},
}

func TestNegotiator(t *testing.T) {
	for i, tc := range negotiateTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: strings.NewReader(tc.in),
				Writer: buf,
			}
			_, err := xmpp.NegotiateSession(context.Background(), tc.location, tc.origin, rw, tc.negotiator)
			if ((err == nil || tc.err == nil) && (err != nil || tc.err != nil)) || err.Error() != tc.err.Error() {
				t.Errorf("Unexpected error: want=%q, got=%q", tc.err, err)
			}
			if out := buf.String(); out != tc.out {
				t.Errorf("Unexpected output:\nwant=%q,\n got=%q", tc.out, out)
			}
		})
	}
}
