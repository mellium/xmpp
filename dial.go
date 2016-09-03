// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"net"
	"strconv"
	"time"

	"mellium.im/xmpp/jid"
)

// A Dialer contains options for connecting to an XMPP address.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is therefore equivalent to just calling
// the Dial function.
type Dialer struct {
	net.Dialer

	// NoLookup stops the dialer from looking up SRV or TXT records for the given
	// domain. It also prevents fetching of the host metadata file.
	// Instead, it will try to connect to the domain directly.
	NoLookup bool
}

// Dial discovers and connects to the address on the named network that services
// the given local address with a client-to-server (c2s) connection.
//
// laddr is the clients origin address. The remote address is taken from the
// origins domain part or from the domains SRV records. For a description of the
// ctx and network arguments, see the Dial function.
func Dial(ctx context.Context, network string, laddr *jid.JID) (*Conn, error) {
	var d Dialer
	return d.Dial(ctx, network, laddr)
}

// DialConfig connects to the address on the named network using the provided
// config.
//
// The context must be non-nil. If the context expires before the connection is
// complete, an error is returned. Once successfully connected, any expiration
// of the context will not affect the connection.
//
// Network may be any of the network types supported by net.Dial, but you almost
// certainly want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
func DialConfig(ctx context.Context, network string, config *Config) (*Conn, error) {
	var d Dialer
	return d.DialConfig(ctx, network, config)
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

// Dial discovers and connects to the address on the named network that services
// the given local address with a client-to-server (c2s) connection.
//
// For a description of the arguments see the Dial function.
func (d *Dialer) Dial(ctx context.Context, network string, laddr *jid.JID) (*Conn, error) {
	c := NewClientConfig(laddr)
	return d.DialConfig(ctx, network, c)
}

// DialConfig connects to the address on the named network using the provided
// config.
//
// For a description of the arguments see the Dial function.
func (d *Dialer) DialConfig(ctx context.Context, network string, config *Config) (*Conn, error) {
	c, err := d.dial(ctx, network, config)
	if err != nil {
		return c, err
	}

	return c, err
}

func (d *Dialer) dial(ctx context.Context, network string, config *Config) (*Conn, error) {
	if ctx == nil {
		panic("xmpp.Dial: nil context")
	}

	// If we haven't specified any stream features, set some default ones.
	// if config.Features == nil || len(config.Features) == 0 {
	// 	stls := StartTLS(config.TLSConfig != nil)
	// 	bind := BindResource()
	// 	username, password := config.Origin.Domain().String(), config.Secret
	// 	sasl := SASL(
	// 		sasl.Plain("",           username, "password"),
	// 		sasl.ScramSha256("",     username, "password"),
	// 		sasl.ScramSha256Plus("", username, "password"),
	// 		sasl.ScramSha1("",       username, "password"),
	// 		sasl.ScramSha1Plus("",   username, "password"),
	// 	)
	// 	config.Features = map[xml.Name]StreamFeature{
	// 		stls.Name: stls,
	// 		sasl.Name: sasl,
	// 		bind.Name: bind,
	// 	}
	// }

	if d.NoLookup {
		p, err := lookupPort(network, connType(config.S2S))
		if err != nil {
			return nil, err
		}
		conn, err := d.Dialer.DialContext(ctx, network, net.JoinHostPort(
			config.Location.Domainpart(),
			strconv.FormatUint(uint64(p), 10),
		))
		if err != nil {
			return nil, err
		}
		return NewConn(ctx, config, conn)
	}

	addrs, err := lookupService(connType(config.S2S), network, config.Location)
	if err != nil {
		return nil, err
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	for _, addr := range addrs {
		if conn, e := d.Dialer.DialContext(
			ctx, network, net.JoinHostPort(
				addr.Target, strconv.FormatUint(uint64(addr.Port), 10),
			),
		); e != nil {
			err = e
			continue
		} else {
			return NewConn(ctx, config, conn)
		}
	}
	return nil, err
}
