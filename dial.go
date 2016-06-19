// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"net"
	"time"

	"golang.org/x/net/context"
	"mellium.im/xmpp/jid"
)

// DialClient discovers and connects to the address on the named network that
// services the given local address with a client-to-server (c2s) connection.
//
// laddr is the clients origin address. The remote address is taken from the
// origins domain part or from the domains SRV records. For a description of the
// ctx and network arguments, see the Dial function.
func DialClient(ctx context.Context, network string, laddr *jid.JID) (*Conn, error) {
	var d Dialer
	return d.DialClient(ctx, network, laddr)
}

// DialServer connects to the address on the named network with a
// server-to-server (s2s) connection.
//
// raddr is the remote servers address and laddr is the local servers origin
// address. For a description of the ctx and network arguments, see the Dial
// function.
func DialServer(ctx context.Context, network string, raddr, laddr *jid.JID) (*Conn, error) {
	var d Dialer
	return d.DialServer(ctx, network, raddr, laddr)
}

// Dial connects to the address on the named network using the provided config.
//
// The context must be non-nil. If the context expires before the connection is
// complete, an error is returned. Once successfully connected, any expiration
// of the context will not affect the connection.
//
// Network may be any of the network types supported by net.Dial, but you almost
// certainly want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
func Dial(ctx context.Context, network string, config *Config) (*Conn, error) {
	var d Dialer
	return d.Dial(ctx, network, config)
}

// A Dialer contains options for connecting to an XMPP address.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is therefore equivalent to just calling
// the DialClient function.
type Dialer struct {
	net.Dialer

	// NoLookup stops the dialer from looking up SRV or TXT records for the given
	// domain. It also prevents fetching of the host metadata file.
	// Instead, it will try to connect to the domain directly.
	NoLookup bool
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

// DialClient discovers and connects to the address on the named network that
// services the given local address with a client-to-server (c2s) connection.
//
// For a description of the arguments see the DialClient function.
func (d *Dialer) DialClient(ctx context.Context, network string, laddr *jid.JID) (*Conn, error) {
	c := NewClientConfig(laddr)
	return d.Dial(ctx, network, c)
}

// DialServer connects to the address on the named network with a
// server-to-server (s2s) connection.
//
// For a description of the arguments see the DialServer function.
func (d *Dialer) DialServer(ctx context.Context, network string, raddr, laddr *jid.JID) (*Conn, error) {
	c := NewServerConfig(raddr, laddr)
	return d.Dial(ctx, network, c)
}

// Dial connects to the address on the named network using the provided config.
//
// For a description of the arguments see the Dial function.
func (d *Dialer) Dial(
	ctx context.Context, network string, config *Config) (*Conn, error) {
	c, err := d.dial(ctx, network, config)
	if err != nil {
		return c, err
	}
	c.connect(ctx)

	return c, err
}
