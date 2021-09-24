// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpptest provides utilities for XMPP testing.
package xmpptest // import "mellium.im/xmpp/internal/xmpptest"

import (
	"context"
	"io"
	"net"
	"strings"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/ns"
	intstream "mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// NopNegotiator marks the state as ready (by returning state|xmpp.Ready) and
// pops the first token (likely <stream:stream>) but does not perform any
// validation on the token, transmit any data over the wire, or perform any
// other session negotiation.
func NopNegotiator(state xmpp.SessionState, streamNS string) xmpp.Negotiator {
	return func(ctx context.Context, in, out *stream.Info, s *xmpp.Session, data interface{}) (xmpp.SessionState, io.ReadWriter, interface{}, error) {
		// Pop the stream start token.
		rc := s.TokenReader()
		defer rc.Close()

		err := intstream.Expect(ctx, in, rc, s.State()&xmpp.Received == xmpp.Received, false)
		if err != nil {
			return state | xmpp.Ready, nil, nil, err
		}
		out.XMLNS = streamNS
		err = intstream.Send(struct {
			io.Reader
			io.Writer
		}{
			Writer: io.Discard,
		}, out, false, stream.DefaultVersion, "", "example.net", "test@example.net", "123")

		return state | xmpp.Ready, nil, nil, err
	}
}

// NewClientSession returns a new client-to-client XMPP session with the state
// bits set to finalState|xmpp.Ready, the origin JID set to "test@example.net"
// and the location JID set to "example.net".
//
// NewClientSession panics on error for ease of use in testing, where a panic is
// acceptable.
func NewClientSession(finalState xmpp.SessionState, rw io.ReadWriter) *xmpp.Session {
	return newSession(finalState, rw, ns.Client)
}

// NewServerSession is like NewClientSession except that the stream uses the
// server-to-server namespace.
func NewServerSession(finalState xmpp.SessionState, rw io.ReadWriter) *xmpp.Session {
	return newSession(finalState, rw, ns.Server)
}

func newSession(finalState xmpp.SessionState, rw io.ReadWriter, streamNS string) *xmpp.Session {
	location := jid.MustParse("example.net")
	origin := jid.MustParse("test@example.net")

	to, from := origin, location
	if finalState&xmpp.Received == xmpp.Received {
		to, from = from, to
	}

	s, err := xmpp.NewSession(
		context.Background(), location, origin,
		struct {
			io.Reader
			io.Writer
		}{
			Reader: io.MultiReader(
				strings.NewReader(`<stream:stream from="`+from.String()+`" to="`+to.String()+`" id="123" version="1.0" xmlns="`+streamNS+`" xmlns:stream="`+stream.NS+`">`),
				rw,
				strings.NewReader(`</stream:stream>`),
			),
			Writer: rw,
		},
		0,
		NopNegotiator(finalState, streamNS),
	)
	if err != nil {
		panic(err)
	}
	return s
}

// Option is a type for configuring a ClientServer.
type Option func(*ClientServer)

// ClientState configures extra state bits to add to the client session.
func ClientState(state xmpp.SessionState) Option {
	return func(c *ClientServer) {
		c.clientState |= state
	}
}

// ServerState configures extra state bits to add to the server session.
func ServerState(state xmpp.SessionState) Option {
	return func(c *ClientServer) {
		c.serverState |= state
	}
}

// ClientHandler sets up the client side of a ClientServer.
func ClientHandler(handler xmpp.Handler) Option {
	return func(c *ClientServer) {
		c.clientHandler = handler
	}
}

// ClientHandlerFunc sets up the client side of a ClientServer using an
// xmpp.HandlerFunc.
func ClientHandlerFunc(handler xmpp.HandlerFunc) Option {
	return ClientHandler(handler)
}

// ServerHandler sets up the server side of a ClientServer.
func ServerHandler(handler xmpp.Handler) Option {
	return func(c *ClientServer) {
		c.serverHandler = handler
	}
}

// ServerHandlerFunc sets up the server side of a ClientServer using an
// xmpp.HandlerFunc.
func ServerHandlerFunc(handler xmpp.HandlerFunc) Option {
	return ServerHandler(handler)
}

// ClientServer is two coupled xmpp.Session's that can respond to one another in
// tests.
type ClientServer struct {
	Client *xmpp.Session
	Server *xmpp.Session

	clientHandler xmpp.Handler
	serverHandler xmpp.Handler
	clientState   xmpp.SessionState
	serverState   xmpp.SessionState
}

// NewClientServer returns a ClientServer with the client and server goroutines
// started.
// Both serve goroutines are started when NewClientServer is called and shut
// down when the ClientServer is closed.
func NewClientServer(opts ...Option) *ClientServer {
	cs := &ClientServer{
		serverState: xmpp.Received,
	}
	for _, opt := range opts {
		opt(cs)
	}

	clientConn, serverConn := net.Pipe()
	cs.Client = NewClientSession(cs.clientState, clientConn)
	cs.Server = NewServerSession(cs.serverState, serverConn)
	/* #nosec */
	go cs.Client.Serve(cs.clientHandler)
	/* #nosec */
	go cs.Server.Serve(cs.serverHandler)
	return cs
}

// Close calls the client and server sessions' close methods.
func (cs *ClientServer) Close() error {
	err := cs.Client.Close()
	if err != nil {
		return err
	}
	return cs.Server.Close()
}
