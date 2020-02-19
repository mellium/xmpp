// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package dial contains methods and types for dialing XMPP connections.
package dial // import "mellium.im/xmpp/dial"

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"

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
// connect to the domainpart directly if dialing the SRV records fails.
// If Dial returns a net.DNSError, setting NoTLS and then NoLookup and trying
// again may be a good fallback order.
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

func getHostPort(domain, network, service string) (host string, port uint16, err error) {
	host, strPort, splitErr := net.SplitHostPort(domain)
	bigPort, parseErr := strconv.ParseUint(strPort, 10, 16)
	p := uint16(bigPort)
	if splitErr != nil || parseErr != nil {
		// If there is no port in the domain part, pick a default port.
		p, err = discover.LookupPort(network, service)
	}
	// If the error was during splitting return the domain instead of the split
	// host.
	if splitErr != nil {
		return domain, p, err
	}
	return host, p, err
}

func (d *Dialer) dial(ctx context.Context, network string, addr jid.JID) (net.Conn, error) {
	domain := addr.Domainpart()
	service := connType(!d.NoTLS, d.S2S)
	var addrs []*net.SRV
	var err error

	// If we're not looking up SRV records, make up some fake ones.
	if d.NoLookup {
		host, p, err := getHostPort(domain, network, service)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, &net.SRV{
			Target:   host,
			Port:     p,
			Priority: 1,
			Weight:   1,
		})
	} else {
		addrs, err = discover.LookupService(ctx, d.Resolver, service, network, addr)
		if err != nil && err != discover.ErrNoServiceAtAddress {
			return nil, err
		}

		// If we're using TLS also try connecting on the plain records.
		if !d.NoTLS {
			aa, err := discover.LookupService(ctx, d.Resolver, connType(d.NoTLS, d.S2S), network, addr)
			if err != nil && err != discover.ErrNoServiceAtAddress {
				return nil, err
			}
			addrs = append(addrs, aa...)
		}

		// If there aren't any records, try connecting on the main domain.
		if len(addrs) == 0 {
			// If there are no SRV records, use domain and default port.
			host, p, err := getHostPort(domain, network, service)
			if err != nil {
				return nil, err
			}

			addrs = []*net.SRV{{
				Target: host,
				Port:   uint16(p),
			}}
		}
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	for _, addr := range addrs {
		var c net.Conn
		var e error
		if d.NoTLS {
			c, e = d.Dialer.DialContext(ctx, network, net.JoinHostPort(
				addr.Target,
				strconv.FormatUint(uint64(addr.Port), 10),
			))
		} else {
			if d.TLSConfig == nil {
				c, e = tls.DialWithDialer(&d.Dialer, network, net.JoinHostPort(
					addr.Target,
					strconv.FormatUint(uint64(addr.Port), 10),
				), &tls.Config{ServerName: domain})
			} else {
				c, e = tls.DialWithDialer(&d.Dialer, network, net.JoinHostPort(
					addr.Target,
					strconv.FormatUint(uint64(addr.Port), 10),
				), d.TLSConfig)
			}
		}
		if e != nil {
			err = e
			continue
		}

		return c, nil
	}
	return nil, err
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
