// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"time"

	"mellium.im/xmpp/internal/discover"
	"mellium.im/xmpp/jid"
)

// Conn is a net.Conn created for the purpose of establishing an XMPP session.
type Conn struct {
	tlsConn *tls.Conn
	c       net.Conn
	rw      io.ReadWriter
}

// newConn wraps an io.ReadWriter in a Conn.
func newConn(rw io.ReadWriter) *Conn {
	nc := &Conn{rw: rw}

	switch typrw := rw.(type) {
	case *Conn:
		return typrw
	case *tls.Conn:
		nc.tlsConn = typrw
		nc.c = typrw
	case net.Conn:
		nc.c = typrw
	}

	return nc
}

// ConnectionState returns basic TLS details about the connection if TLS has
// been negotiated. If TLS has not been negotiated, ok is false.
func (c *Conn) ConnectionState() (connState tls.ConnectionState, ok bool) {
	if c.tlsConn != nil {
		return c.tlsConn.ConnectionState(), true
	}
	return connState, false
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.c.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// Read can be made to time out and return a net.Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (int, error) {
	return c.rw.Write(b)
}

// DialClient discovers and connects to the address on the named network with a
// client-to-server (c2s) connection.
//
// For more information see the Dialer type.
func DialClient(ctx context.Context, network string, addr jid.JID) (*Conn, error) {
	var d Dialer
	return d.Dial(ctx, network, addr)
}

// DialServer discovers and connects to the address on the named network with a
// server-to-server connection (s2s).
//
// For more info see the Dialer type.
func DialServer(ctx context.Context, network string, addr jid.JID) (*Conn, error) {
	d := Dialer{
		S2S: true,
	}
	return d.Dial(ctx, network, addr)
}

// A Dialer contains options for connecting to an XMPP address.
// After a connection is established the Dial method does not attempt to create
// an XMPP session on the connection.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is equivalent to calling the DialClient
// function.
//
// If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not
// affect the connection.
//
// addr is a JID with a domainpart of the server we wish to connect too.
// DialClient will attempt to look up SRV records for the given JIDs domainpart
// or connect to the domainpart directly if no such SRV records exist.
//
// Network may be any of the network types supported by net.Dial, but you almost
// certainly want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
type Dialer struct {
	net.Dialer

	// Resolver allows you to change options related to resolving DNS.
	Resolver *net.Resolver

	// NoLookup stops the dialer from looking up SRV or TXT records for the given
	// domain. It also prevents fetching of the host metadata file.
	// Instead, it will try to connect to the domain directly.
	NoLookup bool

	// S2S causes the server to attempt to dial a server-to-server connection.
	S2S bool
}

// Dial discovers and connects to the address on the named network.
//
// For more information see the Dialer type.
func (d *Dialer) Dial(ctx context.Context, network string, addr jid.JID) (*Conn, error) {
	return d.dial(ctx, network, addr)
}

func (d *Dialer) dial(ctx context.Context, network string, addr jid.JID) (*Conn, error) {
	if d.NoLookup {
		p, err := discover.LookupPort(network, connType(d.S2S))
		if err != nil {
			return nil, err
		}
		c, err := d.Dialer.DialContext(ctx, network, net.JoinHostPort(
			addr.Domainpart(),
			strconv.FormatUint(uint64(p), 10),
		))
		if err != nil {
			return nil, err
		}
		return newConn(c), nil
	}

	addrs, err := discover.LookupService(ctx, d.Resolver, connType(d.S2S), network, addr)
	if err != nil {
		return nil, err
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	for _, addr := range addrs {
		conn, e := d.Dialer.DialContext(
			ctx, network, net.JoinHostPort(
				addr.Target, strconv.FormatUint(uint64(addr.Port), 10),
			),
		)
		if e != nil {
			err = e
			continue
		}

		return newConn(conn), nil
	}
	return nil, err
}

// Copied from the net package in the standard library. Copyright The Go
// Authors.
func minNonzeroTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.Before(b) {
		return a
	}
	return b
}

// Copied from the net package in the standard library. Copyright The Go
// Authors.
//
// deadline returns the earliest of:
//   - now+Timeout
//   - d.Deadline
//   - the context's deadline
// Or zero, if none of Timeout, Deadline, or context's deadline is set.
func (d *Dialer) deadline(ctx context.Context, now time.Time) (earliest time.Time) {
	if d.Timeout != 0 { // including negative, for historical reasons
		earliest = now.Add(d.Timeout)
	}
	if d, ok := ctx.Deadline(); ok {
		earliest = minNonzeroTime(earliest, d)
	}
	return minNonzeroTime(earliest, d.Deadline)
}

func connType(s2s bool) string {
	if s2s {
		return "xmpp-server"
	}
	return "xmpp-client"
}
