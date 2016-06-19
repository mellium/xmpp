// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/net/context"
)

// SessionState represents the current state of an XMPP session. For a
// description of each bit, see the various SessionState typed constants.
type SessionState int8

const (
	// Indicates that the underlying connection has been secured. For instance,
	// after STARTTLS has been performed or if an already secure connection is
	// being used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Indicates that the session has been authenticated via SASL.
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

// NewConn attempts to use an existing connection (or any io.ReadWriteCloser) to
// negotiate an XMPP session based on the given config. If the provided context
// is canceled before stream negotiation is complete an error is returned. After
// stream negotiation if the context is canceled it has no effect.
func NewConn(ctx context.Context, config *Config, rwc io.ReadWriteCloser) (*Conn, error) {
	c := &Conn{
		config: config,
		rwc:    rwc,
		e:      xml.NewEncoder(rwc),
		d:      xml.NewDecoder(rwc),
	}
	return c, c.connect(ctx)
}

// A Conn represents an XMPP connection that can perform SRV lookups for a given
// server and connect to the correct ports.
type Conn struct {
	config   *Config
	rwc      io.ReadWriteCloser
	state    SessionState
	received bool
	e        *xml.Encoder
	d        *xml.Decoder
}

func (c *Conn) connect(ctx context.Context) error {
	// TODO(ssw)
	panic("xmpp: connect not yet implemented")
	return nil
}

// Config returns the connections config.
func (c *Conn) Config() *Config {
	return c.config
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.rwc.Read(b)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.rwc.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.rwc.Close()
}

// State returns the current state of the session.
func (c *Conn) State() SessionState {
	return c.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (c *Conn) LocalAddr() net.Addr {
	if c.received {
		return c.config.Location
	}

	return c.config.Origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (c *Conn) RemoteAddr() net.Addr {
	if c.received {
		return c.config.Origin
	}
	return c.config.Location
}

var errSetDeadline = errors.New("xmpp: cannot set deadline: not using a net.Conn")

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations fail with a timeout
// (see type Error) instead of blocking. The deadline applies to all future I/O,
// not just the immediately following call to Read or Write.
//
// An idle timeout can be implemented by repeatedly extending the deadline after
// successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	if conn, ok := c.rwc.(net.Conn); ok {
		return conn.SetDeadline(t)
	}
	return errSetDeadline
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	if conn, ok := c.rwc.(net.Conn); ok {
		return conn.SetReadDeadline(t)
	}
	return errSetDeadline
}

// SetWriteDeadline sets the deadline for future Write calls. Even if write
// times out, it may return n > 0, indicating that some of the data was
// successfully written. A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	if conn, ok := c.rwc.(net.Conn); ok {
		return conn.SetWriteDeadline(t)
	}
	return errSetDeadline
}
