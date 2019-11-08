// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial_test

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"

	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

// TODO: accept DNS connections and returned canned responses so that we can
// check DNS request type made in the tests.

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
		addr:   "example.net", resolved: true,
	},
	10: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "example.net", resolved: true,
	},
	11: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net", resolved: true,
	},

	// DNS with port
	12: {addr: "example.net:123", resolved: true},
	13: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "example.net:123", resolved: true,
	},
	14: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "example.net:123", resolved: true,
	},
	15: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net:123", resolved: true,
	},
}

func TestDial(t *testing.T) {
	for i, tc := range dialTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			server := newServer(t)

			if tc.dialer == nil {
				tc.dialer = &dial.Dialer{}
			}
			tc.dialer.Dialer = server.Dialer()

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
						panic(fmt.Errorf("dial: error closing dialed connection: %v", err))
					}
				}
			}()

			if server.Dialed != tc.socketAddr {
				t.Errorf("Dialed wrong address: want=%q, got=%q", tc.socketAddr, server.Dialed)
			}
			if server.Resolved != tc.resolved {
				t.Errorf("Wrong value for resolved: want=%t, got=%t", tc.resolved, server.Resolved)
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
