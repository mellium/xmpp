// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpptest provides utilities for XMPP testing.
package xmpptest // import "mellium.im/xmpp/internal/xmpptest"

import (
	"context"
	"io"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

// NewSession returns a new XMPP session with the state bits set to
// state|xmpp.Ready.
//
// NewSession panics on error for ease of use in testing, where a panic is
// acceptable.
func NewSession(state xmpp.SessionState, rw io.ReadWriter) *xmpp.Session {
	location := jid.MustParse("example.net")
	origin := jid.MustParse("test@example.net")

	s, err := xmpp.NegotiateSession(
		context.Background(), location, origin, rw,
		func(_ context.Context, _ *xmpp.Session, _ interface{}) (xmpp.SessionState, io.ReadWriter, interface{}, error) {
			return state | xmpp.Ready, nil, nil, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return s
}
