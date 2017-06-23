// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"net"
	"strconv"
	"time"

	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
)

// Conn is a net.Conn created for the purpose of establishing an XMPP session.
type Conn net.Conn

// DialClient discovers and connects to the address on the named network with a
// client-to-server (c2s) connection.
//
// If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not
// affect the connection.
//
// addr is a JID with a domainpart of the server we wish to connect too.
// DialClient will attempt to look up SRV records for the given JIDs domainpart
// or connect to the domainpart directly.
//
// Network may be any of the network types supported by net.Dial, but you almost
// certainly want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
func DialClient(ctx context.Context, network string, addr *jid.JID) (Conn, error) {
	var d Dialer
	return d.Dial(ctx, network, addr)
}

// DialServer discovers and connects to the address on the named network with a
// server-to-server connection (s2s).
//
// For more info see the DialClient function.
func DialServer(ctx context.Context, network string, addr *jid.JID) (Conn, error) {
	d := Dialer{
		S2S: true,
	}
	return d.Dial(ctx, network, addr)
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

	// Attempt to dial a server-to-server connection.
	S2S bool
}

// Dial discovers and connects to the address on the named network.
//
// For a description of the arguments see the DialClient function.
func (d *Dialer) Dial(ctx context.Context, network string, addr *jid.JID) (Conn, error) {
	return d.dial(ctx, network, addr)
}

func (d *Dialer) dial(ctx context.Context, network string, addr *jid.JID) (Conn, error) {
	if d.NoLookup {
		p, err := internal.LookupPort(network, connType(d.S2S))
		if err != nil {
			return nil, err
		}
		return d.Dialer.DialContext(ctx, network, net.JoinHostPort(
			addr.Domainpart(),
			strconv.FormatUint(uint64(p), 10),
		))
	}

	addrs, err := internal.LookupService(connType(d.S2S), network, addr)
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

		return conn, nil
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
