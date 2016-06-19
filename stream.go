// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"fmt"
	"io"
)

// SessionState represents the current state of an XMPP session. For a
// description of each bit, see the various SessionState typed constants.
type SessionState int8

const (
	// Indicates that the underlying connection has been secured. For instance,
	// after STARTTLS has been performed or if a pre-secured connection is being
	// used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Indicates that the session has been authenticated (probably with SASL).
	Authn

	// Indicates that an XMPP resource has been bound.
	Bind

	// Indicates that the session is fully negotiated and that XMPP stanzas may be
	// sent and received.
	Ready

	// Indicates that the session's streams must be restarted. This bit will
	// trigger an automatic restart and will be flipped back to off as soon as the
	// stream is restarted.
	StreamRestartRequired
)

func sendNewStream(w io.Writer, c *Config, id string) error {
	var ns string
	switch c.S2S {
	case true:
		ns = NSServer
	case false:
		ns = NSClient
	}
	if id == "" {
		id = " "
	} else {
		id = ` id='` + id + `' `
	}
	_, err := fmt.Fprintf(w,
		`<stream:stream%sto='%s' from='%s' version='%s' xml:lang='%s' xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
		id,
		c.Location.String(),
		c.Origin.String(),
		c.Version,
		c.Lang,
		ns,
	)
	if err != nil {
		return err
	}

	// Clear the StreamRestartRequired Bit
	if c, ok := w.(*Conn); ok {
		c.state &= ^StreamRestartRequired
	}
	return err
}

func (c *Conn) negotiateStreams(ctx context.Context) error {
	panic("xmpp: Not yet implemented.")
}
