// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
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
		return fmt.Errorf("error parsing address %q: %w", addr, err)
	}
	d := dial.Dialer{}

	server := j.Domainpart()
retry_dial:
	conn, err := d.DialServer(ctx, "tcp", j, server)
	if err != nil {
		return fmt.Errorf("error dialing session: %w", err)
	}

	s, err := xmpp.NewSession(ctx, j.Domain(), j, conn, 0, xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Lang: "en",
			Features: []xmpp.StreamFeature{
				xmpp.BindResource(),
				xmpp.StartTLS(&tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				}),
				xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
			},
			TeeIn:  xmlIn,
			TeeOut: xmlOut,
		}
	}))
	if err != nil {
		if errors.Is(err, stream.SeeOtherHost) {
			server = err.(stream.Error).Content

			logger.Printf("see-other-host: %s", server)
			s.Close()
			conn.Close()
			goto retry_dial
		}
		return fmt.Errorf("error establishing a session: %w", err)
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

	return s.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {

		// This is a workaround for https://mellium.im/issue/196
		// until a cleaner permanent fix is devised (see https://mellium/issue/197)
		d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), t))
		if _, err := d.Token(); err != nil {
			return err
		}

		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local != "message" {
			return nil
		}

		msg := MessageBody{}
		err = d.DecodeElement(&msg, start)
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
