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

// MessageBody is a message stanza that contains a body. It is normally used for
// chat messages.
type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

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
	err = s.Send(context.TODO(), stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		log.Printf("Error sending initial presence: %q", err)
		return
	}

	s.Serve(xmpp.HandlerFunc(func(t xmlstream.DecodeEncoder, start *xml.StartElement) error {
		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local != "message" {
			return nil
		}

		msg := MessageBody{}
		err = t.DecodeElement(&msg, start)
		if err != nil && err != io.EOF {
			log.Printf("Error decoding message: %q", err)
			return nil
		}

		// Don't reflect messages unless they are chat messages and actually have a
		// body.
		// In a real world situation we'd probably want to respond to IQs, at least.
		if msg.Body == "" || msg.Type != stanza.ChatMessage {
			return nil
		}

		reply := MessageBody{
			Message: stanza.Message{
				To: msg.From.Bare(),
			},
			Body: msg.Body,
		}
		log.Printf("Replying to message %q from %s with body %q", msg.ID, reply.To, reply.Body)
		err = t.Encode(reply)
		if err != nil {
			log.Printf("Error responding to message %q: %q", msg.ID, err)
		}
		return nil
	}))
}
