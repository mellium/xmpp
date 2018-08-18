// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"io"
	"net"

	"mellium.im/xmpp/internal"
)

// teeConn is a net.Conn that also copies reads and writes to the provided
// writers.
type teeConn struct {
	net.Conn
	ctx         context.Context
	multiWriter io.Writer
	teeReader   io.Reader
}

// newTeeConn creates a teeConn. If the provided context is canceled, writes
// start passing through to the underlying net.Conn and are no longer copied to
// in and out.
func newTeeConn(ctx context.Context, c net.Conn, in, out io.Writer) teeConn {
	if tc, ok := c.(teeConn); ok {
		return tc
	}

	tc := teeConn{Conn: c, ctx: ctx}
	if in != nil {
		tc.teeReader = io.TeeReader(c, in)
	}
	if out != nil {
		tc.multiWriter = io.MultiWriter(c, out)
	}
	return tc
}

func (tc teeConn) Write(p []byte) (int, error) {
	if tc.multiWriter == nil {
		return tc.Conn.Write(p)
	}
	select {
	case <-tc.ctx.Done():
		tc.multiWriter = nil
		return tc.Conn.Write(p)
	default:
	}
	return tc.multiWriter.Write(p)
}

func (tc teeConn) Read(p []byte) (int, error) {
	if tc.teeReader == nil {
		return tc.Conn.Read(p)
	}
	select {
	case <-tc.ctx.Done():
		tc.teeReader = nil
		return tc.Conn.Read(p)
	default:
	}
	return tc.teeReader.Read(p)
}

// StreamConfig contains options for configuring the default Negotiator.
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// S2S causes the negotiator to negotiate a server-to-server (s2s) connection.
	S2S bool

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature

	// If set a copy of any reads from the session will be written to TeeIn and
	// any writes to the session will be written to TeeOut (similar to the tee(1)
	// command).
	// This can be used to build an "XML console", but users should be careful
	// since this bypasses TLS and could expose passwords and other sensitve data.
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

				s.in.StreamInfo, err = internal.ExpectNewStream(ctx, s.in.d, s.State()&Received == Received)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), cfg.S2S, internal.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), internal.RandomID())
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), cfg.S2S, internal.DefaultVersion, cfg.Lang, s.location.String(), s.origin.String(), "")
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
				s.in.StreamInfo, err = internal.ExpectNewStream(ctx, s.in.d, s.State()&Received == Received)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
			}
		}

		// TODO: Check if the first token is a stream error (if so, unmarshal and
		// return, otherwise pass the token into negotiateFeatures).
		mask, rw, err = negotiateFeatures(ctx, s, data == nil, cfg.Features)
		nState.doRestart = rw != nil
		return mask, rw, nState, err
	}
}
