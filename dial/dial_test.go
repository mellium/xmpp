// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"syscall"
	"testing"

	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

var dialTests = [...]struct {
	dialer     *dial.Dialer
	addr       string
	socketAddr string
	resolved   bool
}{
	// IP no port
	0: {addr: "::1", socketAddr: "[::1]:5222"},
	1: {
		dialer:     &dial.Dialer{NoLookup: true},
		addr:       "::1",
		socketAddr: "[::1]:5222",
	},
	2: {
		dialer:     &dial.Dialer{NoLookup: false, S2S: true},
		addr:       "::1",
		socketAddr: "[::1]:5269",
	},
	3: {
		dialer:     &dial.Dialer{NoLookup: true, S2S: true},
		addr:       "::1",
		socketAddr: "[::1]:5269",
	},

	// IP with port
	4: {addr: "[::1]:123", socketAddr: "[::1]:123"},
	5: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "[::1]:123", socketAddr: "[::1]:123",
	},
	6: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "[::1]:123", socketAddr: "[::1]:123",
	},
	7: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "[::1]:123", socketAddr: "[::1]:123",
	},

	// DNS no port
	8: {addr: "example.net", resolved: true},
	9: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "example.net", socketAddr: "127.0.0.1:5222",
	},
	10: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "example.net", resolved: true,
	},
	11: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net", socketAddr: "127.0.0.1:5269",
	},

	// DNS with port
	12: {addr: "example.net:123", socketAddr: "127.0.0.1:123"},
	13: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "example.net:123", socketAddr: "127.0.0.1:123",
	},
	14: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "example.net:123", socketAddr: "127.0.0.1:123",
	},
	15: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net:123", socketAddr: "127.0.0.1:123",
	},
}

func TestDial(t *testing.T) {
	for i, tc := range dialTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			server := &Server{}

			if tc.dialer == nil {
				tc.dialer = &dial.Dialer{}
			}
			tc.dialer.Dialer = server.Dialer()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			addr := jid.MustParse(tc.addr)

			conn, err := tc.dialer.Dial(ctx, "tcp", addr)
			// An error will always be encountered, so just log it, don't fail.
			if err != nil && err != io.EOF {
				t.Logf("Error dialing: %v", err)
			}
			defer func() {
				if conn != nil {
					err := conn.Close()
					if err != nil {
						panic(err)
					}
				}
			}()

			if server.Dialed != tc.socketAddr {
				t.Errorf("Dialed wrong address: want=%q, got=%q", tc.socketAddr, server.Dialed)
			}
			if server.Resolved != tc.resolved {
				t.Errorf("Resolved DNS wrong number of times: want=%t, got=%t", tc.resolved, server.Resolved)
			}
		})
	}
}

func TestDialClientPanicsIfNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Dial to panic when passed a nil context.")
		}
	}()
	dial.Client(nil, "tcp", jid.MustParse("feste@shakespeare.lit"))
}

func TestDialServerPanicsIfNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Dial to panic when passed a nil context.")
		}
	}()
	dial.Server(nil, "tcp", jid.MustParse("feste@shakespeare.lit"))
}

// Server listens for service discovery and connection attempts and records the
// types of requests that were made.
type Server struct {
	Dialed   string
	Resolved bool
}

// resolver returns a DNS resolver that uses Go's built in DNS resolver if
// possible and always returns canned requests pointing to the current server.
func (s *Server) resolver() *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			s.Resolved = true
			a, b := net.Pipe()
			/* #nosec */
			a.Close()
			/* #nosec */
			b.Close()
			return a, nil
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
			return errors.New("Expected error: preventing dial")
		},
	}
}
