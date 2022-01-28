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
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
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

	mucClient := &muc.Client{}

	// Handle incoming messages.
	go func() {
		m := mux.New(
			stanza.NSClient,
			mux.Message(stanza.ChatMessage, xml.Name{Local: "body"}, receiveMessage(logger)),
			mux.Message(stanza.GroupChatMessage, xml.Name{Local: "body"}, receiveMessage(logger)),
			muc.HandleClient(mucClient),
		)
		err := session.Serve(m)
		if err != nil {
			logger.Fatalf("Error handling incoming messages: %v", err)
		}
	}()

	printHelp()

	userInput := bufio.NewScanner(os.Stdin)
	mucs := make(map[string]*muc.Channel)
	var mucsM sync.Mutex
	var parsedToAddr jid.JID
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

		const (
			joinPrefix = "join:"
			partPrefix = "part:"
		)
		switch {
		case msg == "help":
			printHelp()
			continue
		case strings.HasPrefix(msg, joinPrefix):
			joinJID, err := jid.Parse(strings.TrimSpace(msg[len(joinPrefix):]))
			if err != nil {
				logger.Printf("error parsing MUC address to join: %v", err)
				continue
			}
			c, err := mucClient.Join(context.TODO(), joinJID, session, muc.MaxHistory(0))
			if err != nil {
				logger.Printf("error joining %s: %v", joinJID, err)
				continue
			}
			mucsM.Lock()
			mucs[joinJID.Bare().String()] = c
			mucsM.Unlock()

			parsedToAddr = joinJID.Bare()
			continue
		case strings.HasPrefix(msg, partPrefix):
			partJID, err := jid.Parse(strings.TrimSpace(msg[len(partPrefix):]))
			if err != nil {
				logger.Printf("error parsing MUC address to join: %v", err)
				continue
			}
			bare := partJID.Bare().String()
			mucsM.Lock()
			c, ok := mucs[bare]
			mucsM.Unlock()
			if !ok {
				logger.Printf("channel %s is not joined", partJID)
				continue
			}
			err = c.Leave(context.TODO(), "")
			if err != nil {
				logger.Printf("failed to leave channel %s: %v", partJID, err)
				continue
			}
			mucsM.Lock()
			delete(mucs, bare)
			mucsM.Unlock()

			if partJID.Equal(parsedToAddr) {
				parsedToAddr = jid.JID{}
			}
			continue
		}

		idx := strings.IndexByte(msg, ':')
		if idx != -1 {
			parsedToAddr, err = jid.Parse(msg[:idx])
			if err != nil {
				logger.Printf("error parsing address: %v", err)
				continue
			}
		}

		if parsedToAddr.Equal(jid.JID{}) {
			printHelp()
			continue
		}

		msg = strings.TrimSpace(msg[idx+1:])

		mucsM.Lock()
		_, ok := mucs[parsedToAddr.Bare().String()]
		mucsM.Unlock()
		stanzaType := stanza.ChatMessage
		if ok {
			stanzaType = stanza.GroupChatMessage
		}
		err = session.Encode(ctx, messageBody{
			Message: stanza.Message{
				To:   parsedToAddr,
				From: parsedAddr,
				Type: stanzaType,
			},
			Body: msg,
		})
		if err != nil {
			logger.Fatalf("error sending message: %v", err)
		}
	}
	if err := userInput.Err(); err != nil {
		logger.Fatalf("error reading user input: %v", err)
	}
}

func printHelp() {
	fmt.Println(`Enter a JID, a colon, and a message to send. For example:

	me@example.net: Test message

Afterwards any future messages you type will be sent to the same JID until you
enter a new one. You can also send messages to multi-user chat (MUC) channels:

	join: foo@channels.example.net/nick
	part: foo@channels.example.net

Bare messages sent after joining a channel will go to the channel.
`)
}

func receiveMessage(logger *log.Logger) mux.MessageHandlerFunc {
	return func(m stanza.Message, t xmlstream.TokenReadEncoder) error {
		d := xml.NewTokenDecoder(t)
		from := m.From
		if m.Type != stanza.GroupChatMessage {
			from = m.From.Bare()
		}

		msg := messageBody{}
		err := d.Decode(&msg)
		if err != nil && err != io.EOF {
			logger.Printf("error decoding message: %q", err)
			return nil
		}

		if msg.Body != "" {
			if msg.Subject != "" {
				fmt.Printf("\nFrom %s: [%s] %s\n"+prompt, from, msg.Subject, msg.Body)
			} else {
				fmt.Printf("\nFrom %s: %q\n"+prompt, from, msg.Body)
			}
		}
		return nil
	}
}
