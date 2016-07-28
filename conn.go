// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmpp/jid"
)

// A Conn represents an XMPP connection that can perform SRV lookups for a given
// server and connect to the correct ports.
type Conn struct {
	config *Config
	rwc    io.ReadWriteCloser
	state  SessionState

	// The actual origin of this conn (we don't want to mutate the config, so if
	// this origin exists and is different from the one in config, eg. because the
	// server did not assign us the resourcepart we requested, this is canonical).
	origin *jid.JID

	// The stream features advertised for the current streams.
	features map[xml.Name]struct{}
	flock    sync.Mutex

	in struct {
		stream
		d *xml.Decoder
	}
	out struct {
		stream
		e *xml.Encoder
	}
}

// Features returns a set of the currently available stream features (including
// those that have already been negotiated).
func (c *Conn) Features() map[xml.Name]struct{} {
	c.flock.Lock()
	defer c.flock.Unlock()

	return c.features
}

// NewConn attempts to use an existing connection (or any io.ReadWriteCloser) to
// negotiate an XMPP session based on the given config. If the provided context
// is canceled before stream negotiation is complete an error is returned. After
// stream negotiation if the context is canceled it has no effect.
func NewConn(ctx context.Context, config *Config, rwc io.ReadWriteCloser) (*Conn, error) {
	c := &Conn{
		config: config,
		rwc:    rwc,
		state:  StreamRestartRequired,
	}

	return c, c.negotiateStreams(ctx)
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
	if (c.state & Received) == Received {
		return c.config.Location
	}
	if c.origin != nil {
		return c.origin
	}
	return c.config.Origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (c *Conn) RemoteAddr() net.Addr {
	if (c.state & Received) == Received {
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
