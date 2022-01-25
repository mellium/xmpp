// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The msgrepl command listens on the given JID and prints and sends messages.
//
// For more information try running the command and typing "help" at the prompt.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	prompt  = "> "
	envAddr = "XMPP_ADDR"
	envPass = "XMPP_PASS"
)

// messageBody is a message stanza that contains a body. It is normally used for
// chat messages.
type messageBody struct {
	stanza.Message
	Subject string `xml:"subject,omitempty"`
	Body    string `xml:"body"`
}

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Get and parse the XMPP address to send from.
	addr := os.Getenv(envAddr)
	if addr == "" {
		logger.Fatalf("Environment variable $%s unset", envAddr)
	}

	parsedAddr, err := jid.Parse(addr)
	if err != nil {
		logger.Fatalf("Error parsing address %q: %v", addr, err)
	}

	// Get the password to use when logging in.
	pass := os.Getenv(envPass)
	if pass == "" {
		logger.Fatalf("Environment variable $%s unset", envPass)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Login to the XMPP server.
	logger.Println("Logging in…")
	dialCtx, dialCtxCancel := context.WithTimeout(ctx, 30*time.Second)
	session, err := xmpp.DialClientSession(
		dialCtx, parsedAddr,
		xmpp.BindResource(),
		xmpp.StartTLS(&tls.Config{
			ServerName: parsedAddr.Domain().String(),
			MinVersion: tls.VersionTLS12,
		}),
		xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
	)
	dialCtxCancel()
	if err != nil {
		logger.Fatalf("Error loging in: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			logger.Fatalf("Error ending session: %v", err)
		}
		if err := session.Conn().Close(); err != nil {
			logger.Fatalf("Error closing connection: %v", err)
		}
	}()

	// Send initial presence to let the server know we want to receive messages.
	err = session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		logger.Fatalf("Error sending initial presence: %v", err)
	}

	// Handle incoming messages.
	go func() {
		err := session.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {

			// This is a workaround for https://github.com/mellium/xmpp/issues/196
			// until a cleaner permanent fix is devised (see https://github.com/mellium/xmpp/issues/197)
			d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), t))
			if _, err := d.Token(); err != nil {
				return err
			}

			// Ignore anything that's not a message. In a real system we'd want to at
			// least respond to IQs.
			if start.Name.Local != "message" {
				return nil
			}

			msg := messageBody{}
			err = d.DecodeElement(&msg, start)
			if err != nil && err != io.EOF {
				logger.Printf("Error decoding message: %q", err)
				return nil
			}

			if msg.Body != "" {
				if msg.Subject != "" {
					fmt.Printf("\nFrom %s: [%s] %s\n"+prompt, msg.From.Bare(), msg.Subject, msg.Body)
				} else {
					fmt.Printf("\nFrom %s: %q\n"+prompt, msg.From.Bare(), msg.Body)
				}
			}
			return nil
		}))
		if err != nil {
			logger.Fatalf("Error handling incoming messages: %v", err)
		}
	}()

	printHelp()

	userInput := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(prompt)
		scanned := userInput.Scan()
		if !scanned {
			break
		}

		msg := userInput.Text()
		if msg == "" {
			continue
		}

		if msg == "help" {
			printHelp()
			continue
		}

		idx := strings.IndexByte(msg, ':')
		if idx == -1 {
			printHelp()
			continue
		}

		parsedToAddr, err := jid.Parse(msg[:idx])
		if err != nil {
			logger.Printf("Error parsing address: %v", err)
			continue
		}

		msg = strings.TrimSpace(msg[idx+1:])

		err = session.Encode(ctx, messageBody{
			Message: stanza.Message{
				To:   parsedToAddr,
				From: parsedAddr,
				Type: stanza.ChatMessage,
			},
			Body: msg,
		})
		if err != nil {
			logger.Fatalf("Error sending message: %v", err)
		}
	}
	if err := userInput.Err(); err != nil {
		logger.Fatalf("Error reading user input: %v", err)
	}
}

func printHelp() {
	fmt.Println("Enter a JID, a colon, and a message to send. eg. me@example.net: Test message")
}
