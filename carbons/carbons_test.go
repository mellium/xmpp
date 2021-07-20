// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package carbons_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)

func TestEnableDisable(t *testing.T) {
	var out bytes.Buffer
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			e := xml.NewEncoder(&out)
			err := e.EncodeToken(*start)
			if err != nil {
				return err
			}
			_, err = xmlstream.Copy(e, t)
			if err != nil {
				return err
			}
			return e.Flush()
		}),
	)
	err := carbons.EnableIQ(context.Background(), cs.Client, stanza.IQ{
		Type: stanza.GetIQ,
		ID:   "000",
	})
	if !errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable}) {
		t.Fatalf("unexpected error enabling carbons: %v", err)
	}
	err = carbons.DisableIQ(context.Background(), cs.Client, stanza.IQ{
		Type: stanza.GetIQ,
		ID:   "000",
	})
	if !errors.Is(err, stanza.Error{Condition: stanza.ServiceUnavailable}) {
		t.Fatalf("unexpected error disabling carbons: %v", err)
	}

	output := out.String()
	const expected = `<iq xmlns="jabber:client" xmlns="jabber:client" type="set" id="000"><enable xmlns="urn:xmpp:carbons:2" xmlns="urn:xmpp:carbons:2"></enable></iq><iq xmlns="jabber:client" xmlns="jabber:client" type="set" id="000"><disable xmlns="urn:xmpp:carbons:2" xmlns="urn:xmpp:carbons:2"></disable></iq>`
	if output != expected {
		t.Errorf("wrong XML:\nwant=%s,\n got=%s", expected, output)
	}
}
