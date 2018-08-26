// Copyright 2017 The Mellium Contributors.
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

// NopNegotiator marks the state as ready (by returning state|xmpp.Ready) and
// does not actually transmit any data over the wire or perform any other
// session negotiation.
func NopNegotiator(state xmpp.SessionState) xmpp.Negotiator {
	return func(_ context.Context, _ *xmpp.Session, _ interface{}) (xmpp.SessionState, io.ReadWriter, interface{}, error) {
		return state | xmpp.Ready, nil, nil, nil
	}
}

// NewSession returns a new XMPP session with the state bits set to
// state|xmpp.Ready, the origin JID set to "test@example.net" and the location
// JID set to "example.net".
//
// NewSession panics on error for ease of use in testing, where a panic is
// acceptable.
func NewSession(state xmpp.SessionState, rw io.ReadWriter) *xmpp.Session {
	location := jid.MustParse("example.net")
	origin := jid.MustParse("test@example.net")

	s, err := xmpp.NegotiateSession(
		context.Background(), location, origin, rw,
		NopNegotiator(state),
	)
	if err != nil {
		panic(err)
	}
	return s
}
