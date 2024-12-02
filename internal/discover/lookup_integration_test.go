// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

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
	resolver  *net.Resolver
	service   string
	addr      jid.JID
	noService bool
	addrs     []*net.SRV
	err       error
}{
	0: {
		err: discover.ErrInvalidService,
	},
	1: {
		service: "xmpp-client",
		addr:    jid.MustParse("me@example.net"),
	},
	2: {
		service: "xmpp-client",
		addr:    testConvJID,
		addrs: []*net.SRV{
			{
				Target:   "xmpp.conversations.im.",
				Port:     5222,
				Priority: 5,
				Weight:   0,
			},
			{
				Target:   "xmpps.conversations.im.",
				Port:     80,
				Priority: 10,
				Weight:   0,
			},
		},
	},
	3: {
		service: "xmpp-server",
		addr:    testConvJID,
		addrs: []*net.SRV{
			{
				Target:   "xmpp.conversations.im.",
				Port:     5269,
				Priority: 5,
				Weight:   0,
			},
		},
	},
	4: {
		service: "xmpp-server",
		addr:    jid.MustParse("samwhited.com"),
		addrs: []*net.SRV{
			{
				Target:   "xmpp-hosting.conversations.im.",
				Port:     5269,
				Priority: 1,
				Weight:   1,
			},
		},
	},
	5: {
		service:   "xmpp-server",
		addr:      jid.MustParse("example@no-service.badxmpp.eu"),
		noService: true,
	},
}

func TestIntegrationLookupService(t *testing.T) {
	for i, tc := range lookupTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			addrs, noService, err := discover.LookupService(context.Background(), tc.resolver, tc.service, tc.addr)
			if noService != tc.noService {
				t.Errorf("unexpected value for '.' record: want=%t, got=%t", tc.noService, noService)
			}
			switch dnsErr := err.(type) {
			case nil:
				if err != tc.err {
					t.Errorf("got unexpected error: want=%q, got=%q", tc.err, err)
				}
			case *net.DNSError:
				var errStr string
				if tc.err != nil {
					errStr = tc.err.Error()
				}
				if dnsErr.Err != errStr {
					t.Errorf("got unexpected error: want=%q, got=%q", errStr, dnsErr.Error())
				}
			default:
				if err != tc.err {
					t.Errorf("got unexpected error: want=%q, got=%q", tc.err, err)
				}
			}
			if len(tc.addrs) != len(addrs) {
				for _, addr := range addrs {
					t.Logf("got addr: %+v", addr)
				}
				t.Fatalf("unexpected addrs: want=%d, got=%d", len(tc.addrs), len(addrs))
			}
			for i, addr := range tc.addrs {
				if *addr != *addrs[i] {
					t.Fatalf("unexpected addr at %d: want=%v, got=%v", i, *addr, *addrs[i])
				}
			}
		})
	}
}
