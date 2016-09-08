// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"net"

	"mellium.im/xmpp/jid"
)

// Listener is an XMPP network listener.
type Listener struct {
	net.Listener

	config *Config
	origin *jid.JID
}

// Listen announces on the local network address laddr. The network must be one
// of the stream-oriented networks supported by net.Listen.
func Listen(network, laddr string, origin *jid.JID, config *Config) (*Listener, error) {
	l, err := net.Listen(network, laddr)
	return &Listener{Listener: l, origin: origin, config: config}, err
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	c := &Session{conn: conn, state: Received, config: l.config}

	// If the connection is a tls.Conn already, make sure we don't advertise
	// StartTLS.
	if _, ok := conn.(*tls.Conn); ok {
		c.state |= Secure
	}

	return c, c.negotiateStreams(context.TODO(), conn)
}

// AcceptXMPP accepts the next incoming call and returns the new connection.
func (l *Listener) AcceptXMPP() (*Session, error) {
	c, err := l.Accept()
	return c.(*Session), err
}

func (l *Listener) Addr() net.Addr {
	return l.origin
}
