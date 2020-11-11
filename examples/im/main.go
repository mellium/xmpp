// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The im command sends XMPP (Jabber) messages from the command line.
// It can send instant messages to individuals and multi-user chats (MUCs),
// similar to mail(1) for SMTP (email).
//
// For more information run im -help.
package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	envAddr = "XMPP_ADDR"
	envPass = "XMPP_PASS"
)

// messageBody is a message stanza that contains a body. It is normally used for
// chat messages.
type messageBody struct {
	stanza.Message
	Subject string `xml:"subject"`
	Thread  string `xml:"thread"`
	Body    string `xml:"body"`
}

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)

	// Get and parse the XMPP address to send from.
	addr := os.Getenv(envAddr)
	if addr == "" {
		logger.Fatalf("environment variable $%s unset", envAddr)
	}

	parsedAddr, err := jid.Parse(addr)
	if err != nil {
		logger.Fatalf("error parsing address %q: %v", addr, err)
	}

	// Get the password to use when logging in.
	pass := os.Getenv(envPass)
	if pass == "" {
		logger.Fatalf("environment variable $%s unset", envPass)
	}

	var (
		help    bool
		rawXML  bool
		room    bool
		uri     bool
		verbose bool
		subject string
	)
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.BoolVar(&help, "help", help, "Show this help message")
	flags.BoolVar(&help, "h", help, "")
	flags.BoolVar(&rawXML, "xml", rawXML, "Treat the input as raw XML to be sent on the stream.")
	flags.BoolVar(&room, "room", room, "The provided JID is a multi-user chat (MUC) room.")
	flags.BoolVar(&uri, "uri", uri, "Parse the recipient as an XMPP URI instead of a JID.")
	flags.BoolVar(&verbose, "v", verbose, "Show verbose logging.")
	flags.StringVar(&addr, "addr", addr, "The XMPP address to connect to, overrides $XMPP_ADDR")
	flags.StringVar(&subject, "subject", subject, "Set the subject of the message or chat room.")

	err = flags.Parse(os.Args[1:])
	switch err {
	case flag.ErrHelp:
		// The -h and -help flags are special cased by flags for some reason and
		// exit even if you don't register them. This should never be hit (since we
		// do register them), but handle the error just in case.
		help = true
	case nil:
	default:
		logger.Fatalf("error parsing flags: %v", err)
	}

	// If the help flag was set, just show the help message and exit.
	if help {
		printHelp(flags)
		os.Exit(0)
	}

	args := flags.Args()
	if len(args) < 1 {
		printHelp(flags)
		os.Exit(1)
	}

	var parsedToAddr jid.JID
	if uri {
		logger.Fatalf("parsing as a URI is not yet implemented")
	} else {
		// Parse the recipient address as a JID.
		parsedToAddr, err = jid.Parse(args[0])
		if err != nil {
			logger.Fatalf("error parsing %q as a JID: %v", args[0], err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Login to the XMPP server.
	debug.Println("logging in…")
	dialCtx, dialCtxCancel := context.WithTimeout(ctx, 30*time.Second)
	session, err := xmpp.DialClientSession(
		dialCtx, parsedAddr,
		xmpp.BindResource(),
		xmpp.StartTLS(&tls.Config{
			ServerName: parsedAddr.Domain().String(),
		}),
		xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1, sasl.Plain),
	)
	dialCtxCancel()
	if err != nil {
		logger.Fatalf("error loging in: %v", err)
	}

	originJID := session.LocalAddr()

	defer func() {
		if room {
			debug.Printf("leaving the chat room %s…", addr)
			err = session.Encode(stanza.Presence{
				ID:   "456def",
				From: originJID,
				To:   parsedToAddr,
				Type: stanza.UnavailablePresence,
			})
			if err != nil {
				logger.Fatalf("error leaving the chat room %s: %v", addr, err)
			}
		}
		if err := session.Close(); err != nil {
			logger.Fatalf("error ending session: %v", err)
		}
		if err := session.Conn().Close(); err != nil {
			logger.Fatalf("error closing connection: %v", err)
		}
	}()

	rawMsg, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		logger.Fatalf("error reading message from stdin: %v", err)
	}
	msg := strings.ToValidUTF8(string(rawMsg), "")

	if room {
		debug.Printf("joining the chat room %s…", addr)
		// Join the MUC.
		joinPresence := struct {
			stanza.Presence
			// TODO: why is the xmlns double encoded, resulting in a broken stanza?
			//
			//X struct {
			//	History struct {
			//		MaxStanzas int `xml:"maxstanzas,attr"`
			//	} `xml:"history"`
			//} `xml:"http://jabber.org/protocol/muc x"`
		}{
			Presence: stanza.Presence{
				ID:   "123abc",
				From: originJID,
				To:   parsedToAddr,
			},
		}
		err = session.Encode(joinPresence)
		if err != nil {
			log.Fatalf("error joining MUC %s: %v", addr, err)
		}
	}

	// Send message
	if rawXML {
		err = session.Send(ctx, xml.NewDecoder(strings.NewReader(msg)))
		if err != nil {
			logger.Fatalf("error sending raw XML: %v", err)
		}
	} else {
		err = session.Encode(messageBody{
			Message: stanza.Message{
				To:   parsedToAddr,
				From: parsedAddr,
				Type: stanza.ChatMessage,
			},
			Body:    msg,
			Subject: subject,
		})
		if err != nil {
			logger.Fatalf("error sending message: %v", err)
		}
	}
}

func printHelp(flags *flag.FlagSet) {
	fmt.Fprintf(flags.Output(), "Usage of %s:\n", os.Args[0])
	flags.PrintDefaults()
	fmt.Printf(`
The im command sends XMPP (Jabber) messages from the command line.
It can send instant messages to individuals and multi-user chats (MUCs),
similar to mail(1) for SMTP (email).

The message will be read from stdin, and all messages will be converted to valid
UTF-8. Invalid byte sequences will be removed.

To configure the command, the following environment variables (shown with their
current value) may be set:

    XMPP_ADDR=%s
    XMPP_PASS=<not shown>
`, os.Getenv(envAddr))
}
