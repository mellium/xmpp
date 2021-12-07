// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package dial contains methods and types for dialing XMPP connections.
package dial // import "mellium.im/xmpp/dial"

import (
	"context"
	"crypto/tls"
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
// an XMPP session on the connection.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is equivalent to calling the Client
// function.
type Dialer struct {
	net.Dialer

	// NoLookup stops the dialer from looking up SRV or TXT records for the given
	// domain. It also prevents fetching of the host metadata file.
	// Instead, it will try to connect to the domain directly.
	NoLookup bool

	// S2S causes the server to attempt to dial a server-to-server connection.
	S2S bool

	// Disable TLS entirely (eg. when using StartTLS on a server that does not
	// support implicit TLS).
	NoTLS bool

	// Attempt to create a TLS connection by first looking up SRV records (unless
	// NoLookup is set) and then attempting to use the domains A or AAAA record.
	// The nil value is interpreted as a tls.Config with the expected host set to
	// that of the connection addresses domain part.
	TLSConfig *tls.Config
}

// Dial discovers and connects to the address on the named network.
// It will attempt to look up SRV records for the JIDs domainpart or
// connect to the domainpart directly if dialing the SRV records fails or is
// disabled.
//
// If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not
// affect the connection.
//
// Network may be any of the network types supported by net.Dial, but you most
// likely want to use one of the tcp connection types ("tcp", "tcp4", or
// "tcp6").
func (d *Dialer) Dial(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	return d.dial(ctx, network, addr)
}

func (d *Dialer) dial(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	domain := addr.Domainpart()
	cfg := d.TLSConfig
	if cfg == nil {
		cfg = &tls.Config{
			ServerName: domain,
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
		return d.legacy(ctx, network, addr, cfg)
	}

	var xmppAddrs, xmppsAddrs []*net.SRV
	var xmppErr, xmppsErr error
	var wg sync.WaitGroup
	wg.Add(1)
	if !d.NoTLS {
		wg.Add(1)
		go func() {
			// Lookup xmpps-(client|server)
			defer wg.Done()
			xmppsService := connType(true, d.S2S)
			addrs, err := discover.LookupService(ctx, d.Resolver, xmppsService, addr)
			if err != nil {
				xmppsErr = err
			}
			xmppsAddrs = addrs
		}()
	}
	go func() {
		// Lookup xmpp-(client|server)
		defer wg.Done()
		xmppService := connType(false, d.S2S)
		addrs, err := discover.LookupService(ctx, d.Resolver, xmppService, addr)
		if err != nil {
			xmppErr = err
		}
		xmppAddrs = addrs
	}()
	wg.Wait()

	// If both lookups failed, return one of the errors.
	if xmppsErr != nil && xmppErr != nil {
		return nil, xmppsErr
	}
	addrs := append(xmppsAddrs, xmppAddrs...)
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no xmpp service found at address %s", domain)
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	var err error
	for _, addr := range addrs {
		var c net.Conn
		var e error
		if d.NoTLS {
			c, e = d.Dialer.DialContext(ctx, network, net.JoinHostPort(
				addr.Target,
				strconv.FormatUint(uint64(addr.Port), 10),
			))
		} else {
			c, e = tls.DialWithDialer(&d.Dialer, network, net.JoinHostPort(
				addr.Target,
				strconv.FormatUint(uint64(addr.Port), 10),
			), cfg)
		}
		if e != nil {
			err = e
			continue
		}

		return c, nil
	}
	return nil, err
}

func (d *Dialer) legacy(ctx context.Context, network string, addr jid.JID, cfg *tls.Config) (net.Conn, error) {
	domain := addr.Domainpart()
	if !d.NoTLS {
		conn, err := tls.DialWithDialer(&d.Dialer, network,
			net.JoinHostPort(domain, "5223"), cfg)
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
