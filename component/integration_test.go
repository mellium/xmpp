// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package component_test

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"testing"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/component"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const (
	domain = "component.localhost"
	secret = "fo0b4r"
)

func TestIntegrationComponentClient(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
		prosody.Component(domain, secret, ""),
	)
	prosodyRun(integrationComponentClient)
}

func integrationComponentClient(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j := jid.MustParse(domain)
	conn, err := cmd.ComponentConn(ctx)
	if err != nil {
		t.Errorf("error dialing connection: %v", err)
	}
	session, err := component.NewSession(ctx, j, []byte(secret), conn)
	if err != nil {
		t.Errorf("error negotiating session: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	go func() {
		defer cancel()
		err := session.Serve(mux.New(
			session.In().XMLNS,
			mux.MessageFunc(stanza.ChatMessage, xml.Name{Local: "body"}, func(_ stanza.Message, r xmlstream.TokenReadEncoder) error {
				defer cancel()
				d := xml.NewTokenDecoder(r)
				msg := struct {
					XMLName xml.Name
					stanza.Message
					Body string `xml:"body"`
				}{}
				err = d.Decode(&msg)
				if err != nil {
					t.Fatalf("error decoding message: %v", err)
				}
				return nil
			}),
		))
		if err != nil {
			t.Errorf("error serving session: %v", err)
		}
	}()

	j, pass := cmd.User()
	client, err := cmd.DialClient(ctx, j, t,
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		}),
		xmpp.SASL("", pass, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		t.Fatalf("error dialing user session: %v", err)
	}
	client.SendMessageElement(ctx, xmlstream.Wrap(
		xmlstream.Token(xml.CharData("test")),
		xml.StartElement{Name: xml.Name{Local: "body"}},
	), stanza.Message{
		To:   jid.MustParse("me@" + domain),
		Type: stanza.ChatMessage,
	})
	<-ctx.Done()
	if err := ctx.Err(); err != nil && err != context.Canceled {
		t.Fatalf("test canceled by context: %v", err)
	}
	// We must close the connection to ensure that the test does not end before
	// the goroutine has a chance to report any errors from serving the session.
	err = session.Close()
	if err != nil {
		t.Fatalf("errror closing component connection: %v", err)
	}
}
