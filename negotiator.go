// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"io"

	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/stream"
)

// Negotiator is a function that can be passed to NegotiateSession to perform
// custom session negotiation. This can be used for creating custom stream
// initialization logic that does not use XMPP feature negotiation such as the
// connection mechanism described in XEP-0114: Jabber Component Protocol.
// Normally NewClientSession or NewServerSession should be used instead.
//
// If a Negotiator is passed into NegotiateSession it will be called repeatedly
// until a mask is returned with the Ready bit set. Each time Negotiator is
// called any bits set in the state mask that it returns will be set on the
// session state and any cache value that is returned will be passed back in
// during the next iteration. If a new io.ReadWriter is returned, it is set as
// the session's underlying io.ReadWriter and the internal session state
// (encoders, decoders, etc.) will be reset.
type Negotiator func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, cache interface{}, err error)

// StreamConfig contains options for configuring the default Negotiator.
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// S2S causes the negotiator to negotiate a server-to-server (s2s) connection.
	S2S bool

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature

	// WebSocket indicates that the negotiator should use the WebSocket
	// subprotocol defined in RFC 7395.
	WebSocket bool

	// Secure marks the connection as secure, even if TLS has not been negotiated.
	// This is useful when a reverse proxy is handling TLS and we can't tell that
	// the connection has already been secured.
	Secure bool

	// If set a copy of any reads from the session will be written to TeeIn and
	// any writes to the session will be written to TeeOut (similar to the tee(1)
	// command).
	// This can be used to build an "XML console", but users should be careful
	// since this bypasses TLS and could expose passwords and other sensitive
	// data.
	TeeIn, TeeOut io.Writer
}

// NewNegotiator creates a Negotiator that uses a collection of StreamFeatures
// to negotiate an XMPP client-to-server (c2s) or server-to-server (s2s)
// session.
// If StartTLS is one of the supported stream features, the Negotiator attempts
// to negotiate it whether the server advertises support or not.
func NewNegotiator(cfg StreamConfig) Negotiator {
	return negotiator(cfg)
}

type negotiatorState struct {
	doRestart bool
	cancelTee context.CancelFunc
}

func negotiator(cfg StreamConfig) Negotiator {
	return func(ctx context.Context, s *Session, data interface{}) (mask SessionState, rw io.ReadWriter, restartNext interface{}, err error) {
		nState, ok := data.(negotiatorState)
		// If no state was passed in, this is the first negotiate call so make up a
		// default.
		if !ok {
			nState = negotiatorState{
				doRestart: true,
				cancelTee: nil,
			}
			if cfg.S2S {
				s.state |= S2S
			}
			if cfg.Secure {
				s.state |= Secure
			}
		}

		c := s.Conn()
		// If the session is not already using a tee conn, but we're configured to
		// use one, return the new teeConn and don't set any state bits.
		if _, ok := c.(teeConn); !ok && (cfg.TeeIn != nil || cfg.TeeOut != nil) {
			// Cancel any previous teeConn's so that we don't double write to in and
			// out.
			if nState.cancelTee != nil {
				nState.cancelTee()
			}

			// This context is just for canceling the tee effect so it is not part of
			// the normal context chain and its parent is Background.
			ctx, cancel := context.WithCancel(context.Background())
			c = newTeeConn(ctx, c, cfg.TeeIn, cfg.TeeOut)
			nState.cancelTee = cancel
			return mask, c, nState, err
		}

		// Loop for as long as we're not done negotiating features or a stream
		// restart is still required.
		if nState.doRestart {
			if (s.state & Received) == Received {
				// If we're the receiving entity wait for a new stream, then send one in
				// response.

				s.in.Info, err = stream.Expect(ctx, s.in.d, s.State()&Received == Received, cfg.WebSocket)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
				s.out.Info, err = stream.Send(s.Conn(), cfg.S2S, cfg.WebSocket, stream.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), attr.RandomID())
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.

				s.out.Info, err = stream.Send(s.Conn(), cfg.S2S, cfg.WebSocket, stream.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), "")
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
				s.in.Info, err = stream.Expect(ctx, s.in.d, s.State()&Received == Received, cfg.WebSocket)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
			}
		}

		mask, rw, err = negotiateFeatures(ctx, s, data == nil, cfg.WebSocket, cfg.Features)
		nState.doRestart = rw != nil
		return mask, rw, nState, err
	}
}
