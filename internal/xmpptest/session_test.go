// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)

func TestNewSession(t *testing.T) {
	state := xmpp.Secure | xmpp.InputStreamClosed
	buf := new(bytes.Buffer)
	s := xmpptest.NewSession(state, buf)

	if mask := s.State(); mask != state|xmpp.Ready {
		t.Errorf("Got invalid state value: want=%d, got=%d", state, mask)
	}

	if out := buf.String(); out != "" {
		t.Errorf("Buffer wrote unexpected tokens: `%s'", out)
	}
}

func TestNewClient(t *testing.T) {
	state := xmpp.Secure
	s := xmpptest.NewClientServer(state, xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		iq, err := stanza.NewIQ(*start)
		if err != nil {
			panic(err)
		}
		r := iq.Result(nil)
		_, err = xmlstream.Copy(t, r)
		return err
	}))
	origIQ := struct {
		stanza.IQ
	}{
		IQ: stanza.IQ{
			ID: "123",
		},
	}
	resp, err := s.EncodeIQ(context.Background(), origIQ)
	if err != nil {
		t.Errorf("error encoding IQ: %v", err)
	}
	iq := stanza.IQ{}
	err = xml.NewTokenDecoder(resp).Decode(&iq)
	if err != nil {
		t.Errorf("error decoding response: %v", err)
	}
	if iq.ID != origIQ.ID {
		t.Errorf("Response IQ had wrong ID: want=%s, got=%s", origIQ.ID, iq.ID)
	}
}
