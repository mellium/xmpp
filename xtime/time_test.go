// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xtime_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"regexp"
	"strings"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/xtime"
)

var (
	_ xml.Marshaler       = xtime.Time{}
	_ xml.Unmarshaler     = (*xtime.Time)(nil)
	_ xmlstream.Marshaler = xtime.Time{}
	_ xmlstream.WriterTo  = xtime.Time{}
)

type tokenReadEncoder struct {
	xml.TokenReader
	xmlstream.Encoder
}

func TestRoundTrip(t *testing.T) {
	// TODO: this test will likely be shared between all IQ handler packages. Can
	// we provide a helper in xmpptest to automate it?
	var req bytes.Buffer
	s := xmpptest.NewSession(0, &req)
	to := jid.MustParse("to@example.net")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := xtime.Get(ctx, s, to)
	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	h := xtime.Handler{
		TimeFunc: func() time.Time {
			return time.Time{}
		},
	}

	d := xml.NewDecoder(strings.NewReader(req.String()))
	d.DefaultSpace = ns.Server
	tok, _ := d.Token()
	start := tok.(xml.StartElement)
	var b strings.Builder
	e := xml.NewEncoder(&b)

	m := mux.New(xtime.Handle(h))
	err = m.HandleXMPP(tokenReadEncoder{
		TokenReader: d,
		Encoder:     e,
	}, &start)
	if err != nil {
		t.Errorf("unexpected error handling ping: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("unexpected error flushing encoder: %v", err)
	}

	out := b.String()
	// TODO: figure out a better way to ignore randomly generated IDs.
	out = regexp.MustCompile(`id=".*?"`).ReplaceAllString(out, `id="123"`)
	const expected = `<iq type="result" from="to@example.net" id="123"><time xmlns="urn:xmpp:time"><tzo>Z</tzo><utc>0001-01-01T00:00:00Z</utc></time></iq>`
	if out != expected {
		t.Errorf("got=%s, want=%s", out, expected)
	}
}
