// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"log"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	login = "echo@example.net"
	pass  = "just an example don't hardcode passwords"
)

func Example_echobot() {
	j := jid.MustParse(login)
	s, err := xmpp.DialClientSession(
		context.TODO(), j,
		xmpp.BindResource(),
		xmpp.StartTLS(true, &tls.Config{
			ServerName: j.Domain().String(),
		}),
		xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
	)
	if err != nil {
		log.Printf("Error establishing a session: %q", err)
		return
	}
	defer func() {
		log.Println("Closing session…")
		if err := s.Close(); err != nil {
			log.Printf("Error closing session: %q", err)
		}
		log.Println("Closing conn…")
		if err := s.Conn().Close(); err != nil {
			log.Printf("Error closing connection: %q", err)
		}
	}()

	// Send initial presence to let the server know we want to receive messages.
	err = s.Send(context.TODO(), stanza.WrapPresence(jid.JID{}, stanza.AvailablePresence, nil))
	if err != nil {
		log.Printf("Error sending initial presence: %q", err)
		return
	}

	s.Serve(xmpp.HandlerFunc(func(s *xmpp.Session, start *xml.StartElement) error {
		r := s.TokenReader()
		defer r.Close()
		d := xml.NewTokenDecoder(r)

		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local != "message" {
			return nil
		}

		msg := struct {
			stanza.Message
			Body string `xml:"body"`
		}{}
		err = d.DecodeElement(&msg, start)
		if err != nil && err != io.EOF {
			log.Printf("Error decoding message: %q", err)
			return nil
		}

		// Don't reflect messages unless they are chat messages and actually have a
		// body.
		if msg.Body == "" || msg.Type != stanza.ChatMessage {
			return nil
		}

		reply := stanza.WrapMessage(
			msg.From.Bare(), stanza.ChatMessage,
			xmlstream.Wrap(xmlstream.ReaderFunc(func() (xml.Token, error) {
				return xml.CharData(msg.Body), io.EOF
			}), xml.StartElement{Name: xml.Name{Local: "body"}}),
		)
		err = s.Send(context.TODO(), reply)
		if err != nil {
			log.Printf("Error responding to message %q: %q", msg.ID, err)
		}
		return nil
	}))
}
