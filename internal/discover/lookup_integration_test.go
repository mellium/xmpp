// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

package discover_test

import (
	"context"
	"net"
	"strconv"
	"testing"

	"mellium.im/xmpp/internal/discover"
	"mellium.im/xmpp/jid"
)

var testConvJID = jid.MustParse("sam@conversations.im")

var lookupTests = [...]struct {
	resolver *net.Resolver
	service  string
	addr     net.Addr
	addrs    []*net.SRV
	err      error
}{
	0: {},
	1: {
		service: "xmpp-client",
		addr:    jid.MustParse("me@example.net"),
		addrs: []*net.SRV{
			&net.SRV{
				Target: "example.net",
				Port:   5222,
			},
		},
	},
	2: {
		service: "xmpp-client",
		addr:    testConvJID,
		addrs: []*net.SRV{
			&net.SRV{
				Target:   "xmpp.conversations.im.",
				Port:     5222,
				Priority: 5,
				Weight:   1,
			},
		},
	},
	3: {
		service: "xmpp-server",
		addr:    &testConvJID,
		addrs: []*net.SRV{
			&net.SRV{
				Target:   "xmpp.conversations.im.",
				Port:     5269,
				Priority: 5,
				Weight:   1,
			},
		},
	},
	4: {
		service: "xmpp-server",
		addr:    jid.MustParse("samwhited.com"),
		addrs: []*net.SRV{
			&net.SRV{
				Target:   "xmpp-hosting.conversations.im.",
				Port:     5269,
				Priority: 1,
				Weight:   1,
			},
		},
	},
}

func TestLookupService(t *testing.T) {
	for i, tc := range lookupTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			addrs, err := discover.LookupService(context.Background(), tc.resolver, tc.service, "tcp", tc.addr)
			switch dnsErr := err.(type) {
			case nil:
				if err != tc.err {
					t.Errorf("Got unexpected error: want=%q, got=%q", tc.err, err)
				}
			case *net.DNSError:
				var errStr string
				if tc.err != nil {
					errStr = tc.err.Error()
				}
				if dnsErr.Err != errStr {
					t.Errorf("Got unexpected error: want=%q, got=%q", errStr, dnsErr.Error())
				}
			default:
				if err != tc.err {
					t.Errorf("Got unexpected error: want=%q, got=%q", tc.err, err)
				}
			}
			if len(tc.addrs) != len(addrs) {
				t.Fatalf("Unexpected addrs: want=%d, got=%d", len(tc.addrs), len(addrs))
			}
			for i, addr := range tc.addrs {
				if *addr != *addrs[i] {
					t.Fatalf("Unexpected addr at %d: want=%v, got=%v", i, *addr, *addrs[i])
				}
			}
		})
	}
}
