// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpptest provides utilities for XMPP testing.
package xmpptest // import "mellium.im/xmpp/internal/xmpptest"

import (
	"context"
	"io"
	"strings"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// NopNegotiator marks the state as ready (by returning state|xmpp.Ready) and
// pops the first token (likely <stream:stream>) but does not perform any
// validation on the token, transmit any data over the wire, or perform any
// other session negotiation.
func NopNegotiator(state xmpp.SessionState) xmpp.Negotiator {
	return func(ctx context.Context, s *xmpp.Session, data interface{}) (xmpp.SessionState, io.ReadWriter, interface{}, error) {
		// Pop the stream start token.
		rc := s.TokenReader()
		defer rc.Close()

		_, err := rc.Token()
		return state | xmpp.Ready, nil, nil, err
	}
}

// NewSession returns a new client-to-client XMPP session with the state bits
// set to state|xmpp.Ready, the origin JID set to "test@example.net" and the
// location JID set to "example.net".
//
// NewSession panics on error for ease of use in testing, where a panic is
// acceptable.
func NewSession(state xmpp.SessionState, rw io.ReadWriter) *xmpp.Session {
	location := jid.MustParse("example.net")
	origin := jid.MustParse("test@example.net")

	s, err := xmpp.NegotiateSession(
		context.Background(), location, origin,
		struct {
			io.Reader
			io.Writer
		}{
			Reader: io.MultiReader(
				strings.NewReader(`<stream:stream xmlns="`+ns.Client+`" xmlns:stream="`+stream.NS+`">`),
				rw,
				strings.NewReader(`</stream:stream>`),
			),
			Writer: rw,
		},
		false,
		NopNegotiator(state),
	)
	if err != nil {
		panic(err)
	}
	return s
}
