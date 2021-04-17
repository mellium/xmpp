// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest

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
)

type testWriter struct {
	prefix string
	t      *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s: %s", w.prefix, p)
	return len(p), nil
}

// FeatureTestCase is a data driven test for stream feature negotiation.
type FeatureTestCase struct {
	State      xmpp.SessionState
	Feature    xmpp.StreamFeature
	In         string
	Out        string
	FinalState xmpp.SessionState
	Err        error
	ErrStrCmp  bool
}

// RunFeatureTests simulates a stream feature neogtiation and tests the output.
func RunFeatureTests(t *testing.T, tcs []FeatureTestCase) {
	for i, tc := range tcs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var buf bytes.Buffer
			e := xml.NewEncoder(&buf)
			d := xml.NewDecoder(&buf)

			_, err := tc.Feature.List(context.Background(), e, xml.StartElement{
				Name: tc.Feature.Name,
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
			_, data, err := tc.Feature.Parse(context.Background(), d, &start)
			if ok := checkFeatureErr("parse", t, tc, err); ok {
				return
			}

			buf.Reset()
			s := NewSession(tc.State, struct {
				io.Reader
				io.Writer
			}{
				Reader: io.TeeReader(strings.NewReader(tc.In), testWriter{t: t, prefix: "Read"}),
				Writer: io.MultiWriter(testWriter{t: t, prefix: "Sent"}, &buf),
			})
			mask, _, err := tc.Feature.Negotiate(context.Background(), s, data)
			checkFeatureErr("negotiate", t, tc, err)
			if out := buf.String(); out != tc.Out {
				t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.Out, out)
			}
			if tc.FinalState != mask {
				t.Errorf("wrong output state: want=%v, got=%v", tc.FinalState, mask|tc.State)
			}
		})
	}
}

func checkFeatureErr(step string, t *testing.T, tc FeatureTestCase, err error) bool {
	if err == nil {
		return false
	}
	if tc.Err == nil {
		t.Errorf("unexpected error during %s: %v", step, err)
	}
	if tc.ErrStrCmp {
		if err == nil {
			err = errors.New("nil")
		}
		if tc.Err == nil {
			tc.Err = errors.New("nil")
		}
		if err.Error() != tc.Err.Error() {
			t.Errorf("wrong error str during %s: want=%q, got=%q", step, tc.Err, err)
		}
	} else {
		if !errors.Is(err, tc.Err) {
			t.Errorf("wrong error during %s: want=%v, got=%v", step, tc.Err, err)
		}
	}
	return false
}
