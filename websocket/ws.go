// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package websocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/websocket"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/discover"
	"mellium.im/xmpp/jid"
)

// NewSession establishes an XMPP session from the perspective of the initiating
// client on rw using the WebSocket subprotocol.
// It does not perform the WebSocket handshake.
func NewSession(ctx context.Context, addr jid.JID, rw io.ReadWriter, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	wsConn, ok := rw.(*websocket.Conn)
	n := xmpp.NewNegotiator(xmpp.StreamConfig{
		Features:  features,
		WebSocket: true,
		Secure:    ok && wsConn.LocalAddr().(*websocket.Addr).Scheme == "wss",
	})
	return xmpp.NewSession(ctx, addr.Domain(), addr, rw, n)
}

// ReceiveSession establishes an XMPP session from the perspective of the
// receiving server on rw using the WebSocket subprotocol.
// It does not perform the WebSocket handshake.
func ReceiveSession(ctx context.Context, rw io.ReadWriter, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	wsConn, ok := rw.(*websocket.Conn)
	n := xmpp.NewNegotiator(xmpp.StreamConfig{
		Features:  features,
		WebSocket: true,
		Secure:    ok && wsConn.LocalAddr().(*websocket.Addr).Scheme == "wss",
	})
	return xmpp.ReceiveSession(ctx, rw, n)
}

// NewClient performs the WebSocket handshake on rwc and then attempts to
// establish an XMPP session on top of it.
// Location is the WebSocket location and addr is the actual JID expected at the
// XMPP layer.
func NewClient(ctx context.Context, origin, location string, addr jid.JID, rwc io.ReadWriteCloser, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	d := Dialer{
		Origin: origin,
	}
	cfg, err := d.config(location)
	if err != nil {
		return nil, err
	}
	conn, err := websocket.NewClient(cfg, rwc)
	if err != nil {
		return nil, err
	}
	return NewSession(ctx, addr, conn, features...)
}

// DialSession uses a default dialer to create a WebSocket connection and
// attempts to negotiate an XMPP session over it.
//
// If the provided context is canceled after stream negotiation is complete it
// has no effect on the session.
func DialSession(ctx context.Context, origin string, addr jid.JID, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	conn, err := Dial(ctx, origin, addr)
	if err != nil {
		return nil, err
	}
	return NewSession(ctx, addr, conn, features...)
}

// Dial discovers WebSocket endpoints associated with the given address and
// attempts to make a connection to one of them (with appropriate fallback
// behavior).
//
// Calling Dial is the equivalent of creating a Dialer type with only the Origin
// option set and calling its Dial method.
func Dial(ctx context.Context, origin string, addr jid.JID) (net.Conn, error) {
	d := Dialer{
		Origin: origin,
	}
	return d.Dial(ctx, addr)
}

// DialDirect dials the provided WebSocket endpoint without performing any TXT
// or Web Host Metadata file lookup.
//
// Calling DialDirect is the equivalent of creating a Dialer type with only the
// Origin option set and calling its DialDirect method.
func DialDirect(ctx context.Context, origin, addr string) (net.Conn, error) {
	d := Dialer{
		Origin: origin,
	}
	return d.DialDirect(ctx, addr)
}

// Dialer discovers and connects to the WebSocket address on the named network.
// The zero value for each field is equivalent to dialing without that option
// with the exception of Origin (which is required).
// Dialing with the zero value of Dialer (except Origin) is equivalent to
// calling the Dial function.
type Dialer struct {
	// A WebSocket client origin.
	Origin string

	// TLS config for secure WebSocket (wss).
	// If TLSConfig is nil a default config is used.
	TLSConfig *tls.Config

	// Allow falling back to insecure WebSocket connections without TLS.
	// If endpoint discovery is used and a secure WebSocket endpoint is available
	// it will still be prioritized.
	//
	// The WebSocket transport does not support StartTLS so this value will fall
	// back to using bare WebSockets (a scheme of ws:) and is therefore insecure
	// and should never be used.
	InsecureNoTLS bool

	// Additional header fields to be sent in WebSocket opening handshake.
	Header http.Header

	// Dialer used when opening websocket connections.
	Dialer *net.Dialer

	// Resolver to use when looking up TXT records.
	Resolver *net.Resolver

	// HTTP Client to use when looking up Web Host Metadata files.
	Client *http.Client
}

// Dial opens a new client connection to a WebSocket.
//
// If addr is a hostname or has a scheme of "http" or "https" it will attempt to
// look up TXT records and Web Host Metadata files to find WebSocket endpoints
// to connect to.
// If however addr is a complete URI with a scheme of "ws" or "wss" it will
// attempt to connect to the provided endpoint directly with no other lookup.
func (d *Dialer) Dial(ctx context.Context, addr jid.JID) (net.Conn, error) {
	// Setup defaults for the underlying client and resolver.
	httpClient := d.Client
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	netResolver := d.Resolver
	if netResolver == nil {
		netResolver = &net.Resolver{}
	}

	urls, err := discover.LookupWebSocket(ctx, netResolver, httpClient, addr)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("websocket: no XMPP websocket endpoint found on %s", addr.Domainpart())
	}
	// Prioritize wss over anything else, then ws, then anything else that will
	// likely just result in an error.
	sort.Slice(urls, func(i, j int) bool {
		switch {
		case strings.HasPrefix(urls[i], "wss:"):
			return true
		case strings.HasPrefix(urls[i], "wss:"):
			return false
		case strings.HasPrefix(urls[i], "ws:"):
			return true
		}
		return false
	})

	var conn net.Conn
	var cfg *websocket.Config
	for _, u := range urls {
		if !d.InsecureNoTLS && strings.HasPrefix(u, "ws:") {
			continue
		}
		cfg, err = d.config(u)
		if err != nil {
			continue
		}
		conn, err = websocket.DialConfig(cfg)
		if err == nil {
			return conn, err
		}
	}
	return conn, err
}

// DialDirect dials the websocket endpoint without performing any TXT or Web
// Host Metadata file lookup.
//
// Context is currently not used due to restrictions in the underlying WebSocket
// implementation.
// This may change in the future.
func (d *Dialer) DialDirect(_ context.Context, addr string) (net.Conn, error) {
	cfg, err := d.config(addr)
	if err != nil {
		return nil, err
	}
	return websocket.DialConfig(cfg)
}

func (d *Dialer) config(addr string) (cfg *websocket.Config, err error) {
	cfg, err = websocket.NewConfig(addr, d.Origin)
	if err != nil {
		return nil, err
	}
	cfg.Protocol = []string{WSProtocol}
	cfg.TlsConfig = d.TLSConfig
	if cfg.TlsConfig == nil {
		cfg.TlsConfig = &tls.Config{
			ServerName: cfg.Location.Host,
		}
	}
	cfg.Dialer = d.Dialer
	return cfg, nil
}
