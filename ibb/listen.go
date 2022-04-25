// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"context"
	"errors"
	"net"
	"sync"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

type expected struct {
	c      chan *Conn
	cancel context.CancelFunc
}

// Listener is an implementation of net.Listener that is used to accept
// incoming IBB streams.
type Listener struct {
	s        *xmpp.Session
	h        *Handler
	c        chan *Conn
	expected map[string]expected
	eLock    sync.Mutex
}

// Accept waits for the next incoming IBB stream and returns the connection.
// If the listener is closed by either end pending Accept calls unblock and
// return an error.
func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.c
	if !ok {
		return nil, errors.New("ibb: accept on closed listener")
	}
	return conn, nil
}

// Expect is like Accept except that it accepts a specific session that has been
// negotiated out-of-band.
// If Accept and Expect are both waiting on connections, Expect will take
// precedence.
// If Expect is called twice for the same session the original call will be
// canceled and return a context error and the new Expect call will take over.
func (l *Listener) Expect(ctx context.Context, from jid.JID, sid string) (net.Conn, error) {
	l.eLock.Lock()
	if l.expected == nil {
		l.expected = make(map[string]expected)
	}
	key := from.String() + ":" + sid
	e, ok := l.expected[key]
	if ok {
		e.cancel()
	}
	e.c = make(chan *Conn)
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	l.expected[key] = e
	l.eLock.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn, ok := <-e.c:
		if !ok {
			return nil, errors.New("ibb: accept on closed listener")
		}
		return conn, nil
	}
}

// Close stops listening and causes any pending Accept calls to unblock and
// return an error.
// Already accepted connections are not closed.
func (l *Listener) Close() error {
	l.h.lM.Lock()
	defer l.h.lM.Unlock()
	delete(l.h.l, l.s.LocalAddr().String())
	close(l.c)
	return nil
}

// Addr returns the local address for which this listener is accepting
// connections.
func (l *Listener) Addr() net.Addr {
	return l.s.LocalAddr()
}
