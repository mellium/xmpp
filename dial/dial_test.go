// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package dial_test

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"testing"

	"golang.org/x/net/dns/dnsmessage"

	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

// TODO: reply with canned DNS responses so that we can check followup queries.

var dialTests = [...]struct {
	dialer       *dial.Dialer
	addr         string
	socketAddr   string
	dnsQuestions []dnsmessage.Question
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
	8: {
		addr: "example.net",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("_xmpps-client._tcp.example.net."),
			Type:  dnsmessage.TypeSRV,
			Class: dnsmessage.ClassINET,
		}},
	},
	9: {
		dialer: &dial.Dialer{S2S: true},
		addr:   "example.org",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("_xmpps-server._tcp.example.org."),
			Type:  dnsmessage.TypeSRV,
			Class: dnsmessage.ClassINET,
		}},
	},
	10: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "example.com",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.com."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},
	11: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.net."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},

	//// DNS with port
	12: {
		addr: "example.net:123",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.net."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},
	13: {
		dialer: &dial.Dialer{NoLookup: true},
		addr:   "example.org:123",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.org."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},
	14: {
		dialer: &dial.Dialer{NoLookup: false, S2S: true},
		addr:   "example.com:123",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.com."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},
	15: {
		dialer: &dial.Dialer{NoLookup: true, S2S: true},
		addr:   "example.net:123",
		dnsQuestions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName("example.net."),
			Type:  dnsmessage.TypeAAAA,
			Class: dnsmessage.ClassINET,
		}},
	},
}

func TestDial(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %d subtests in short mode", len(dialTests))
	}

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

			if !reflect.DeepEqual(server.Questions, tc.dnsQuestions) {
				t.Errorf("Wrong dns questions:\nwant=%v,\n got=%v", tc.dnsQuestions, server.Questions)
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
