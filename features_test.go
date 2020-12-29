// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
)

type testWriter struct {
	prefix string
	t      *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s: %s", w.prefix, p)
	return len(p), nil
}

type featureTestCase struct {
	state      xmpp.SessionState
	sf         xmpp.StreamFeature
	in         string
	out        string
	finalState xmpp.SessionState
	err        error
}

func runFeatureTests(t *testing.T, tcs []featureTestCase) {
	for i, tc := range tcs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var buf bytes.Buffer
			e := xml.NewEncoder(&buf)
			d := xml.NewDecoder(&buf)

			_, err := tc.sf.List(context.Background(), e, xml.StartElement{
				Name: tc.sf.Name,
			})
			if ok := checkFeatureErr("list", t, tc, err); ok {
				return
			}
			err = e.Flush()
			if err != nil {
				t.Fatalf("error flushing listing: %v", err)
			}
			tok, err := d.Token()
			if err != nil {
				t.Fatalf("error popping start token: %v", err)
			}
			start := tok.(xml.StartElement)
			_, data, err := tc.sf.Parse(context.Background(), d, &start)
			if ok := checkFeatureErr("parse", t, tc, err); ok {
				return
			}

			buf.Reset()
			s := xmpptest.NewSession(tc.state, struct {
				io.Reader
				io.Writer
			}{
				Reader: io.TeeReader(strings.NewReader(tc.in), testWriter{t: t, prefix: "Read"}),
				Writer: io.MultiWriter(testWriter{t: t, prefix: "Sent"}, &buf),
			})
			mask, _, err := tc.sf.Negotiate(context.Background(), s, data)
			if ok := checkFeatureErr("negotiate", t, tc, err); ok {
				return
			}
			if out := buf.String(); out != tc.out {
				t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, out)
			}
			if tc.finalState != mask|tc.state {
				t.Errorf("wrong output state: want=%v, got=%v", tc.finalState, mask|tc.state)
			}
		})
	}
}

func checkFeatureErr(step string, t *testing.T, tc featureTestCase, err error) bool {
	if err == nil {
		return false
	}
	if tc.err == nil {
		t.Fatalf("unexpected error during %s: %v", step, err)
	}
	if !errors.Is(err, tc.err) {
		t.Fatalf("wrong error during %s: want=%v, got=%v", step, tc.err, err)
	}
	return true
}
