// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
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

// Sends a new XML header followed by a stream start element on the given
// io.Writer. We don't use an xml.Encoder both because Go's standard library xml
// package really doesn't like the namespaced stream:stream attribute and
// because we can guarantee well-formedness of the XML with a print in this case
// and printing is much faster than encoding. Afterwards, clear the
// StreamRestartRequired bit.
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
	_, err := fmt.Fprint(w, xml.Header)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w,
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

	// Clear the StreamRestartRequired bit
	if c, ok := w.(*Conn); ok {
		c.state &= ^StreamRestartRequired
	}
	return err
}

// Fetch a token from the given decoder. If it is not a new stream start element
// (or an XML header followed by a stream), error. Clear the
// StreamRestartRequired bit afterwards.
func expectNewStream(ctx context.Context, d *xml.Decoder, c *Conn) error {
	var foundHeader bool
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			// TODO: Validate the token and clear the StreamRestartRequired bit.
			panic("xmpp: Not yet implemented.")
			c.state &= ^StreamRestartRequired
		case xml.ProcInst:
			// TODO: If version or encoding are declared, validate XML 1.0 and UTF-8.
			if !foundHeader && tok.Target == "xml" {
				foundHeader = true
				continue
			}
			// TODO: What errors should we use for this? Check the RFC.
			return NotAuthorized
		default:
			// TODO: What errors should we use for this? Check the RFC.
			return NotAuthorized
		}
	}
}

func (c *Conn) negotiateStreams(ctx context.Context) error {
	panic("xmpp: Not yet implemented.")
}
