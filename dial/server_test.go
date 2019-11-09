// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"syscall"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/nettest"
)

// Server listens for service discovery and connection attempts and records the
// types of requests that were made.
type Server struct {
	Dialed    string
	Questions []dnsmessage.Question

	resolved bool
	t        *testing.T
	dns      net.Listener
}

func newServer(t *testing.T) *Server {
	dns, err := nettest.NewLocalListener("unix")
	if err != nil {
		panic(fmt.Errorf("dial_test: error dialing server: %w", err))
	}

	s := &Server{
		t:   t,
		dns: dns,
	}

	go func() {
		for {
			conn, err := dns.Accept()
			if err != nil {
				t.Logf("dial_test: error accepting connection: %v", err)
				return
			}
			defer func() {
				err := conn.Close()
				t.Logf("dial_test: error closing connection: %v", err)
			}()

			s.handleDNS(t, conn)
		}
	}()

	return s
}

// resolver returns a DNS resolver that uses Go's built in DNS resolver if
// possible and always returns canned requests pointing to the current server.
func (s *Server) resolver() *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
			if s.resolved {
				return nil, fmt.Errorf("dial_test: expected error: only resolve once")
			}

			s.resolved = true
			d := &net.Dialer{}
			return d.DialContext(ctx, "unix", s.dns.Addr().String())
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

func (s *Server) handleDNS(t *testing.T, conn net.Conn) {
	buf, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Fatalf("dial_test: error reading DNS request: %v", err)
	}

	var p dnsmessage.Parser
	_, err = p.Start(buf)
	if err != nil {
		t.Fatalf("dial_test: error parsing DNS request header: %v", err)
	}

	// Parse DNS question
	for {
		q, err := p.Question()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			t.Fatalf("dial_test: error parsing DNS question: %v", err)
		}

		// Normalize A lookups to AAAA to prevent test flaky-ness since we don't
		// really care which was used.
		if q.Type == dnsmessage.TypeA {
			q.Type = dnsmessage.TypeAAAA
		}
		s.Questions = append(s.Questions, q)
	}

	// Parse DNS answers
	err = p.SkipAllAnswers()
	if err != nil {
		t.Fatalf("dial_test: error skipping DNS answers: %v", err)
	}
}
