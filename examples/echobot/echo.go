// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// MessageBody is a message stanza that contains a body. It is normally used for
// chat messages.
type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

func echo(ctx context.Context, addr, pass string, xmlIn, xmlOut io.Writer, logger, debug *log.Logger) error {
	j, err := jid.Parse(addr)
	if err != nil {
		return fmt.Errorf("Error parsing address %q: %w", addr, err)
	}

	conn, err := dial.Client(ctx, "tcp", j)
	if err != nil {
		return fmt.Errorf("Error dialing sesion: %w", err)
	}

	s, err := xmpp.NegotiateSession(ctx, j.Domain(), j, conn, false, xmpp.NewNegotiator(xmpp.StreamConfig{
		Lang: "en",
		Features: []xmpp.StreamFeature{
			xmpp.BindResource(),
			xmpp.StartTLS(true, &tls.Config{
				ServerName: j.Domain().String(),
			}),
			xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
		},
		TeeIn:  xmlIn,
		TeeOut: xmlOut,
	}))
	if err != nil {
		return fmt.Errorf("Error establishing a session: %w", err)
	}
	defer func() {
		logger.Println("Closing conn…")
		if err := s.Conn().Close(); err != nil {
			logger.Printf("Error closing connection: %q", err)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			logger.Println("Closing session…")
			if err := s.Close(); err != nil {
				logger.Printf("Error closing session: %q", err)
			}
		}
	}()

	// Send initial presence to let the server know we want to receive messages.
	err = s.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return fmt.Errorf("Error sending initial presence: %w", err)
	}

	return s.Serve(xmpp.HandlerFunc(func(t xmlstream.DecodeEncoder, start *xml.StartElement) error {
		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local != "message" {
			return nil
		}

		msg := MessageBody{}
		err = t.DecodeElement(&msg, start)
		if err != nil && err != io.EOF {
			logger.Printf("Error decoding message: %q", err)
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
		debug.Printf("Replying to message %q from %s with body %q", msg.ID, reply.To, reply.Body)
		err = t.Encode(reply)
		if err != nil {
			logger.Printf("Error responding to message %q: %q", msg.ID, err)
		}
		return nil
	}))
}
