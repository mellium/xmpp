// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"errors"
	"net"

	"mellium.im/xmpp"
)

// Listener is an implementation of net.Listener that is used to accept
// incoming IBB streams.
type Listener struct {
	s *xmpp.Session
	h *Handler
	c chan *Conn
}

// Accept waits for the next incoming IBB stream and returns the connection.
// If the listener is closed by either end pending Accept calls unblock and
// return an error.
func (l Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.c
	if !ok {
		return nil, errors.New("ibb: accept on closed listener")
	}
	return conn, nil
}

// Close stops listening and causes any pending Accept calls to unblock and
// return an error.
// Already accepted connections are not closed.
func (l Listener) Close() error {
	delete(l.h.l, l.s.LocalAddr().String())
	close(l.c)
	return nil
}

// Addr returns the local address for which this listener is accepting
// connections.
func (l Listener) Addr() net.Addr {
	return l.s.LocalAddr()
}
