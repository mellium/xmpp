// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package carbons_test

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

func TestHandler(t *testing.T) {
	wait := make(chan struct{})
	h := carbons.Handler{
		F: func(m stanza.Message, sent bool, inner xml.TokenReader) error {
			defer close(wait)
			if sent {
				t.Error("expected received not to set sent")
			}
			tok, err := inner.Token()
			if err != nil {
				return err
			}
			if start, ok := tok.(xml.StartElement); !ok || start.Name.Local != "message" {
				t.Errorf("inner payload not handled correctly, got %T token: %[1]v", tok)
			}
			return nil
		},
	}
	m := mux.New(ns.Client, carbons.Handle(h))
	s := xmpptest.NewClientServer(xmpptest.ClientHandler(m))
	defer s.Close()
	const recv = `<message xmlns='jabber:client'
         from='romeo@montague.example'
         to='romeo@montague.example/home'
         type='chat'><received xmlns='urn:xmpp:carbons:2'><forwarded xmlns='urn:xmpp:forward:0'><message xmlns='jabber:client' from='juliet@capulet.example/balcony' to='romeo@montague.example/garden' type='chat'><body>What man art thou that, thus bescreen'd in night, so stumblest on my counsel?</body><thread>0e3141cd80894871a68e6fe6b1ec56fa</thread></message></forwarded></received></message>`
	d := xml.NewDecoder(strings.NewReader(recv))
	err := s.Server.Send(context.Background(), d)
	if err != nil {
		t.Fatalf("error sending carbon: %v", err)
	}
	<-wait
}
