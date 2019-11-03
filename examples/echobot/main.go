// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The echobot command listens on the given JID and replies to messages with the
// same contents.
//
// For more information try running:
//
//     echobot -help
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
)

/* #nosec */
const (
	envAddr = "XMPP_ADDR"
	envPass = "XMPP_PASS"
)

type logWriter struct {
	logger *log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.logger.Printf("%s", p)
	return len(p), nil
}

func main() {
	// Setup logging and a verbose logger that's disabled by default.
	logger := log.New(os.Stderr, "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)

	// Configure behavior based on flags and environment variables.
	var (
		addr    = os.Getenv(envAddr)
		verbose bool
		logXML  bool
	)
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\n  $%s: The JID which will be used to listen for messages to echo\n  $%s: The password\n\n", envAddr, envPass)
		flags.PrintDefaults()
	}
	flags.BoolVar(&verbose, "v", verbose, "turns on verbose debug logging")
	flags.BoolVar(&logXML, "vv", logXML, "turns on verbose debug and XML logging")

	switch err := flags.Parse(os.Args[1:]); err {
	case flag.ErrHelp:
		return
	case nil:
	default:
		logger.Fatal(err)
	}

	// Return a sane error if the address is empty instead of erroring out when we
	// try to parse it.
	if addr == "" {
		logger.Fatalf("Address not specified, use the -addr flag or set $%s", envAddr)
	}

	// Enable verbose logging if the flag was set.
	if verbose || logXML {
		debug.SetOutput(os.Stderr)
	}

	// Enable XML logging if the flag was set.
	var xmlIn, xmlOut io.Writer
	if logXML {
		xmlIn = logWriter{log.New(os.Stdout, "IN ", log.LstdFlags)}
		xmlOut = logWriter{log.New(os.Stdout, "OUT ", log.LstdFlags)}
	}

	pass := os.Getenv(envPass)
	if pass == "" {
		debug.Printf("The environment variable $%s is empty", envPass)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and gracefully shut down the bot.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()

	if err := echo(ctx, addr, pass, xmlIn, xmlOut, logger, debug); err != nil {
		logger.Fatal(err)
	}
}
