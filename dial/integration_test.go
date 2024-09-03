// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package dial_test

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

var dialTests = []struct {
	dialer  dial.Dialer
	domain  string
	err     string
	errType error
}{
	{
		dialer:  dial.Dialer{S2S: true},
		domain:  "no-target.badxmpp.eu",
		errType: &net.OpError{},
	},
	{
		dialer:  dial.Dialer{S2S: true},
		domain:  "no-address.badxmpp.eu",
		errType: &net.OpError{},
	},
	{
		dialer: dial.Dialer{S2S: true, NoTLS: true},
		domain: "no-service.badxmpp.eu",
		err:    "no xmpp service found at address no-service.badxmpp.eu",
	},
	{
		// If one service returns "." (no service at this address) try the other
		// (which in this case is still an error).
		dialer:  dial.Dialer{S2S: true},
		domain:  "no-service.badxmpp.eu",
		errType: &net.OpError{},
	},
}

func TestIntegrationDial(t *testing.T) {
	for _, tc := range dialTests {
		t.Run(tc.domain, func(t *testing.T) {
			tries := 3
		retry:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			j := jid.MustParse("test@" + tc.domain)
			_, err := tc.dialer.Dial(ctx, "tcp", j)
			if dnsErr, ok := err.(*net.DNSError); tries > 0 && ok && (dnsErr.Temporary() || dnsErr.Timeout()) {
				tries--
				t.Logf("DNS lookup failed for %s, retries remaining %d: %v", tc.domain, tries, err)
				goto retry
			}
			switch {
			case tc.err != "" && err == nil:
				t.Errorf("expected error if SRV record target is missing, got none")
			case tc.err != "" && !strings.HasSuffix(err.Error(), tc.err):
				t.Errorf("wrong error: want=%s, got=%v", tc.err, err.Error())
			case tc.err == "" && tc.errType != nil:
				if reflect.TypeOf(tc.errType) != reflect.TypeOf(err) {
					t.Errorf("wrong type of error: want=%T(%#v), got=%T(%#v)", tc.errType, tc.errType, err, err)
				}
			case tc.err == "" && err != nil:
				t.Errorf("got unexpected error: %v", err)
			}
		})
	}
}
