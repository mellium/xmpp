// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial_test

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
)

// Server listens for service discovery and connection attempts and records the
// types of requests that were made.
type Server struct {
	Dialed   string
	Resolved bool
}

func newServer(t *testing.T) *Server {
	return &Server{}
}

// resolver returns a DNS resolver that uses Go's built in DNS resolver if
// possible and always returns canned requests pointing to the current server.
func (s *Server) resolver() *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
			s.Resolved = true
			return nil, errors.New("dial_test: expected error: preventing resolver dial")
		},
	}
}

// Dialer creates a new dialer that is configured to use a resolver pointing at
// the current server.
func (s *Server) Dialer() net.Dialer {
	return net.Dialer{
		Resolver: s.resolver(),
		Control: func(network, address string, c syscall.RawConn) error {
			s.Dialed = address
			return errors.New("dial_test: expected error: preventing dial")
		},
	}
}
