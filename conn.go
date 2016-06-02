// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"crypto/tls"
	"log"
	"net"
	"time"

	"mellium.im/xmpp/jid"
)

// A Conn represents an XMPP connection that can perform SRV lookups for a given
// server and connect to the correct ports.
type Conn struct {
	conn net.Conn

	log           *log.Logger
	tlsConfig     *tls.Config
	conntype      string
	srvExpiration time.Duration
	dialer        net.Dialer
	network       string
	raddr         *jid.JID
	laddr         *jid.JID

	// DNS Cache
	cname   string
	addrs   []*net.SRV
	srvtime time.Time
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address as a JID.
func (c *Conn) LocalAddr() net.Addr {
	return c.laddr
}

// RemoteAddr returns the remote network address as a JID.
func (c *Conn) RemoteAddr() net.Addr {
	return c.raddr
}

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
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls. Even if write
// times out, it may return n > 0, indicating that some of the data was
// successfully written. A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
