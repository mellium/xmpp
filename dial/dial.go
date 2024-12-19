// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package dial provides mechanisms to resolve and dial XMPP endpoints.
//
// This package provides advanced configuration for establishing the initial
// network layer sockets that will be used for XMPP session negotiation later.
// For simple uses that do not require extra configuration, users may instead
// use the [mellium.im/xmpp.DialClientSession],
// [mellium.im/xmpp.DialServerSession], or [mellium.im/xmpp.DialSession]
// functions to perform both service discovery and session negotiation at once.
//
// Projects requiring more advanced configuration of the connection options may
// instead choose to use the [Client] and [Server] shortcuts in this package to
// first dial the TCP connection without performing any session negotiation.
// Session negotiation can then be completed using the
// [mellium.im/xmpp.NewClientSession], [mellium.im/xmpp.NewServerSession], or
// [mellium.im/xmpp.NewSession] function.
//
// Projects such as clients requiring the ability to enable or disable service
// discovery, configure implicit TLS, configure timeouts, etc. will want to
// create and configure a [Dialer] instead.
//
// # Service Discovery
//
// An XMPP service is discovered by looking up DNS SRV records from the
// domainpart of the XMPP address.
// For details on the SRV record format see [RFC6120 §3.2.1] and [XEP-0368].
// This package looks up SRV records by default, but their lookup may be
// disabled entirely by creating and configuring a [Dialer].
// If SRV lookup is disabled or no SRV records are found the [Dialer] will
// instead lookup A or AAAA records for the specified hostname or XMPP address
// domainpart and attempt to connect on two default ports:
//
//   - 5223 for implicit TLS (sometimes called "direct TLS") connections
//   - 5222 for plain or opportunistic TLS (STARTTLS) connections
//
// Services that set SRV records and explicitly do not provide implicit TLS
// should indicate this using an xmpps record that points to "." to disable
// implicit TLS connection attempts.
//
// # Implicit TLS
//
// If service discovery returns any XMPP endpoints that provide implicit TLS
// (ie. performing a TLS handshake immediately after dialing the transport layer
// connection), these ports are tried first for safety reasons.
// If the server does not support implicit TLS and no timeout is configured the
// connection may take longer than anticipated, or hang entirely.
// This is especially true for servers using the default ports described above.
// This behavior can be disabled entirely on the [Dialer] (eg. for servers that
// we know only support STARTTLS).
//
// # Timeouts
//
// As with any network connection, timeouts or deadlines should be set to ensure
// that a misconfigured server or bad network hardware can't block the
// connection attempt forever.
// For the simple connection functions in [mellium.im/xmpp] this is accomplished
// by passing a [context.Context] to the function.
// The context cancelation will apply to the entire connection and XMPP session
// negotiation attempt.
// For many uses, this is not granular enough.
// By using this package to first dial the connection, then performing session
// negotiation separately we can have separate timeouts for the transport and
// application layers of our connection.
// However, this package may also make multiple connection attempts per call to
// a dialer function or method (ie. one per SRV record returned until a
// connection is made, or one with implicit TLS and one without as described
// previously in this document).
// For any robust system it is important that we make sure that a single
// connection attempt cannot block until the sole timeout is reached, thus
// preventing any further attempts, so a shorter timeout should be set per
// connection attempt.
// This can be accomplished by configuring the underlying [net.Dialer]:
//
//	Dialer{
//		Dialer: net.Dialer{
//			Timeout: 5*time.Second,
//		},
//		TLSConfig: &tls.Config{…},
//	}
//
// [RFC6120 §3.2.1]: https://datatracker.ietf.org/doc/html/rfc6120#section-3.2.1
// [XEP-0368]: https://xmpp.org/extensions/xep-0368.html
package dial // import "mellium.im/xmpp/dial"

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"mellium.im/xmpp/internal/discover"
	"mellium.im/xmpp/jid"
)

// Client discovers and connects to the address on the named network with a
// client-to-server (c2s) connection.
//
// For more information see the Dialer type.
func Client(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	var d Dialer
	return d.Dial(ctx, network, addr)
}

// Server discovers and connects to the address on the named network with a
// server-to-server connection (s2s).
//
// For more info see the Dialer type.
func Server(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	d := Dialer{
		S2S: true,
	}
	return d.Dial(ctx, network, addr)
}

// A Dialer contains options for connecting to an XMPP address.
// After a connection is established the Dial method does not attempt to create
// an XMPP session on the connection, the various session establishment
// functions in the main xmpp package should be passed the resulting connection.
//
// Dialing with the zero value of Dialer is equivalent to calling the Client
// function.
type Dialer struct {
	net.Dialer

	// NoLookup stops the dialer from looking up SRV records for the given domain.
	// Instead, it will try to connect to the domain directly.
	NoLookup bool

	// S2S causes the server to attempt to dial a server-to-server connection.
	S2S bool

	// Disable implicit TLS entirely (eg. when using opportunistic TLS on a server
	// that does not support implicit TLS).
	NoTLS bool

	// The configuration to use when dialing with implicit TLS support.
	// Setting TLSConfig has no effect if NoTLS is true.
	// The default value is interpreted as a tls.Config with the expected host set
	// to that of the connection addresses domain part.
	TLSConfig *tls.Config
}

// Dial discovers and connects to the address on the named network.
// If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not
// affect the connection.
//
// Network may be any of the network types supported by net.Dial, but you most
// likely want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
func (d *Dialer) Dial(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	return d.dial(ctx, network, addr, addr.Domainpart())
}

// DialServer behaves exactly the same as Dial, besides that the server it tries
// to connect to is given as argument instead of using the domainpart of the JID.
//
// Changing the server does not affect the server name expected by the default
// TLSConfig which remains the addresses domainpart.
func (d *Dialer) DialServer(ctx context.Context, network string, addr jid.JID, server string) (net.Conn, error) {
	return d.dial(ctx, network, addr, server)
}

func (d *Dialer) dial(ctx context.Context, network string, addr jid.JID, server string) (net.Conn, error) {
	cfg := d.TLSConfig
	if cfg == nil {
		cfg = &tls.Config{
			ServerName: addr.Domainpart(),
			MinVersion: tls.VersionTLS12,
		}
		// XEP-0368
		if d.S2S {
			cfg.NextProtos = []string{"xmpp-server"}
		} else {
			cfg.NextProtos = []string{"xmpp-client"}
		}
	}
	// If we're not looking up SRV records, use the A/AAAA fallback.
	if d.NoLookup {
		return d.legacy(ctx, network, server, cfg)
	}

	var (
		xmppAddrs, xmppsAddrs           []*net.SRV
		xmppNotPresent, xmppsNotPresent bool
		xmppErr, xmppsErr               error
		wg                              sync.WaitGroup
		xmppsService                    = connType(true, d.S2S)
		xmppService                     = connType(false, d.S2S)
	)
	wg.Add(1)
	if !d.NoTLS {
		wg.Add(1)
		go func() {
			// Lookup xmpps-(client|server)
			defer wg.Done()
			xmppsAddrs, xmppsNotPresent, xmppsErr = discover.LookupServiceByDomain(ctx, d.Resolver, xmppsService, server)
		}()
	}
	go func() {
		// Lookup xmpp-(client|server)
		defer wg.Done()
		xmppAddrs, xmppNotPresent, xmppErr = discover.LookupServiceByDomain(ctx, d.Resolver, xmppService, server)
	}()
	wg.Wait()

	// Set a fallback if either record set was not present.
	if !d.NoTLS && !xmppsNotPresent && len(xmppsAddrs) == 0 {
		xmppsAddrs = discover.FallbackRecords(xmppsService, server)
	}
	if !xmppNotPresent && len(xmppAddrs) == 0 {
		xmppAddrs = discover.FallbackRecords(xmppService, server)
	}

	// If both lookups failed, return the errors.
	if xmppsErr != nil && xmppErr != nil {
		return nil, errors.Join(xmppsErr, xmppErr)
	}
	addrs := make([]*net.SRV, 0, len(xmppAddrs)+len(xmppsAddrs))
	addrs = append(addrs, xmppsAddrs...)
	addrs = append(addrs, xmppAddrs...)
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no xmpp service found at address %s", server)
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	var err error
	for i, addr := range addrs {
		var c net.Conn
		var e error
		// Do not dial expecting a TLS connection if we're trying addreses that we
		// expect starttls on or if we have implicit TLS disabled.
		if d.NoTLS || i >= len(xmppsAddrs) {
			c, e = d.Dialer.DialContext(ctx, network, net.JoinHostPort(
				addr.Target,
				strconv.FormatUint(uint64(addr.Port), 10),
			))
		} else {
			tlsDialer := &tls.Dialer{
				NetDialer: &d.Dialer,
				Config:    cfg,
			}
			c, e = tlsDialer.DialContext(ctx, network, net.JoinHostPort(
				addr.Target,
				strconv.FormatUint(uint64(addr.Port), 10),
			))
		}
		if e != nil {
			err = e
			continue
		}

		return c, nil
	}
	return nil, err
}

func (d *Dialer) legacy(ctx context.Context, network string, domain string, cfg *tls.Config) (net.Conn, error) {
	if !d.NoTLS {
		tlsDialer := &tls.Dialer{
			NetDialer: &d.Dialer,
			Config:    cfg,
		}
		conn, err := tlsDialer.DialContext(ctx, network,
			net.JoinHostPort(domain, "5223"))
		if err == nil {
			return conn, nil
		}
	}

	return d.Dialer.DialContext(ctx, network, net.JoinHostPort(domain, "5222"))
}

func connType(useTLS, s2s bool) string {
	switch {
	case useTLS && s2s:
		return "xmpps-server"
	case !useTLS && s2s:
		return "xmpp-server"
	case useTLS && !s2s:
		return "xmpps-client"
	}
	return "xmpp-client"
}
