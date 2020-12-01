// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//+build integration

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
		dialer: dial.Dialer{S2S: true},
		domain: "no-target.badxmpp.eu",
		err:    "no such host",
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
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			j := jid.MustParse("test@" + tc.domain)

			_, err := tc.dialer.Dial(ctx, "tcp", j)
			switch {
			case tc.err != "" && err == nil:
				t.Errorf("expected error if SRV record target is missing, got none")
			case tc.err != "" && !strings.HasSuffix(err.Error(), tc.err):
				t.Errorf("wrong error: want=%s, got=%v", tc.err, err.Error())
			case tc.err == "" && tc.errType != nil:
				if reflect.TypeOf(tc.errType) != reflect.TypeOf(err) {
					t.Errorf("wrong type of error: want=%T, got=%T", tc.errType, err)
				}
			case tc.err == "" && err != nil:
				t.Errorf("got unexpected error: %v", err)
			}
		})
	}
}
