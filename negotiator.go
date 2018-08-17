// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"io"

	"mellium.im/xmpp/internal"
)

// StreamConfig contains options for configuring the default Negotiator.
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// S2S causes the negotiator to negotiate a server-to-server (s2s) connection.
	S2S bool

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature
}

// NewNegotiator creates a Negotiator that uses a collection of
// StreamFeatures to negotiate an XMPP server-to-server session.
func NewNegotiator(cfg StreamConfig) Negotiator {
	return negotiator(cfg)
}

func negotiator(cfg StreamConfig) Negotiator {
	return func(ctx context.Context, s *Session, doRestart interface{}) (mask SessionState, rw io.ReadWriter, restartNext interface{}, err error) {
		// Loop for as long as we're not done negotiating features or a stream restart
		// is still required.
		if rst, ok := doRestart.(bool); ok && rst {
			if (s.state & Received) == Received {
				// If we're the receiving entity wait for a new stream, then send one in
				// response.

				s.in.StreamInfo, err = internal.ExpectNewStream(ctx, s.in.d, s.State()&Received == Received)
				if err != nil {
					return mask, nil, false, err
				}
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), cfg.S2S, internal.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), internal.RandomID())
				if err != nil {
					return mask, nil, false, err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), cfg.S2S, internal.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), "")
				if err != nil {
					return mask, nil, false, err
				}
				s.in.StreamInfo, err = internal.ExpectNewStream(ctx, s.in.d, s.State()&Received == Received)
				if err != nil {
					return mask, nil, false, err
				}
			}
		}

		mask, rw, err = negotiateFeatures(ctx, s, cfg.Features)
		return mask, rw, rw != nil, err
	}
}
