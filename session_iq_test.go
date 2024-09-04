// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

func TestResponseToTimedOutIQ(t *testing.T) {
	// Regression test for #399

	ctx, cancel := context.WithCancel(context.Background())
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(toks xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// Cancel the context after the server has started processing the
			// response.
			// Technically I think this may be flakey, but I couldn't think of another
			// way to reproduce the problem and, at least for now, the way the
			// buffering in the ClientServer works it should all be fine, but small
			// changes could break this later.
			cancel()
			iq, err := stanza.NewIQ(*start)
			if err != nil {
				return err
			}
			return ping.Handler{}.HandleIQ(iq, toks, start)
		}),
		xmpptest.ClientHandlerFunc(func(toks xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(xmlstream.Discard(), toks)
			return err
		}),
	)
	/* #nosec */
	defer func() {
		err := cs.Close()
		if err != nil {
			t.Fatalf("error closing client/server: %v", err)
		}
	}()

	_, err := cs.Client.EncodeIQ(ctx, ping.IQ{
		IQ: stanza.IQ{
			Type: stanza.GetIQ,
		},
	})
	if err != context.Canceled {
		t.Fatalf("error encoding IQ: %v", err)
	}
}

func TestInvalidStanzaErrorResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deadline terminated test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Test that stanza responses match on the stanza name as well as ID.
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(tr xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			// If we send an IQ and get a presence with the same ID, we should not
			// match the response.
			start.Name.Local = "presence"
			p, err := stanza.NewPresence(*start)
			if err != nil {
				return err
			}
			p.Type = stanza.ErrorPresence
			p.To, p.From = p.From, p.To
			_, err = xmlstream.Copy(tr, p.Wrap(nil))
			if err != nil {
				return err
			}

			// To terminate the test without relying on potentially flakey timeouts,
			// we send an IQ afterwards (and then verify that it was the IQ that was
			// matched even though they both have the same ID).
			start.Name.Local = "iq"
			iq, err := stanza.NewIQ(*start)
			if err != nil {
				return err
			}
			_, err = xmlstream.Copy(tr, iq.Result(nil))
			return err
		}),
		xmpptest.ClientHandlerFunc(func(tr xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			if start.Name.Local == "iq" {
				// We know things failed, so bail out of the EncodeIQ call so that the
				// test can exit cleanly.
				defer cancel()
				t.Fatal("IQ should not have been handled by general handler")
			}
			_, err := xmlstream.Copy(xmlstream.Discard(), tr)
			return err
		}),
	)

	resp, err := cs.Client.EncodeIQ(ctx, ping.IQ{
		IQ: stanza.IQ{
			ID:   "123",
			Type: stanza.GetIQ,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error encoding IQ: %v", err)
	}
	tok, err := resp.Token()
	if err != nil {
		t.Fatalf("error reading response token: %v", err)
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		t.Fatalf("unexpected token in response: %v", tok)
	}
	iqName := xml.Name{Space: stanza.NSClient, Local: "iq"}
	if start.Name != iqName {
		t.Fatalf("wrong stanza in response: want=%v, got=%v", iqName, start.Name)
	}
}
