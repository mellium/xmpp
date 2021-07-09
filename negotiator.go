// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"fmt"
	"io"

	"mellium.im/xmpp/internal/attr"
	intstream "mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// Negotiator is a function that can be passed to NewSession to perform custom
// session negotiation.
// This can be used for creating custom stream initialization logic that does
// not use XMPP feature negotiation such as the connection mechanism described
// in XEP-0114: Jabber Component Protocol.
//
// If a Negotiator is passed into NewSession it will be called repeatedly until
// a mask is returned with the Ready bit set.
// When the negotiator reads the new stream start element it should unmarshal
// the correct values into "in" and set the correct values in "out" for the
// input and output stream respectively.
// Each time Negotiator is called any bits set in the state mask that it returns
// will be set on the session state, and any cache value that is returned will
// be passed back in during the next iteration.
// If a new io.ReadWriter is returned, it is set as the session's underlying
// io.ReadWriter and the internal session state (encoders, decoders, etc.) will
// be reset.
type Negotiator func(ctx context.Context, in, out *stream.Info, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, cache interface{}, err error)

// StreamConfig contains options for configuring the default Negotiator.
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature

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
//
// The function will be called every time a new stream is started so that the
// user may look up required stream features (and other stream configuration)
// based on information about an incoming stream such as the location and origin
// JID.
// Individual features still control whether or not they are listed at any
// given time, so all possible features should be returned on each step and
// new features only added to the list when we learn that they are possible
// eg. because the origin or location JID is set and we can look up that users
// configuration in the database.
// For example, you would not return StartTLS the first time this function is
// called then return Auth once you see that the secure bit is set on the
// session state because the stream features themselves would handle this for
// you.
// Instead you would always return StartTLS and Auth, but you might only add
// the "password reset" feature once you see that the origin JID is one that
// has a backup email in the database.
// The previous stream config is passed in at each step so that it can be
// re-used or the stream features may be appended to if desired (however, this
// is not required).
func NewNegotiator(cfg func(*Session, StreamConfig) StreamConfig) Negotiator {
	return negotiator(false, cfg)
}

// NewWebSocketNegotiator is like NewNegotiator except that it uses the
// WebSocket subprotocol defined in RFC 7395.
func NewWebSocketNegotiator(cfg func(*Session, StreamConfig) StreamConfig) Negotiator {
	return negotiator(true, cfg)
}

type negotiatorState struct {
	doRestart bool
	cancelTee context.CancelFunc
}

func negotiator(websocket bool, cf func(*Session, StreamConfig) StreamConfig) Negotiator {
	var cfg StreamConfig
	return func(ctx context.Context, in, out *stream.Info, s *Session, data interface{}) (mask SessionState, rw io.ReadWriter, restartNext interface{}, err error) {
		cfg = cf(s, cfg)
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

				location := s.LocalAddr()
				origin := s.RemoteAddr()
				err = intstream.Expect(ctx, in, s.in.d, s.State()&Received == Received, websocket)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}

				switch {
				case s.state&S2S == 0 && origin.Equal(jid.JID{}):
					// If we're a server receiving a c2s connection and "from" wasn't
					// previously set, just set it as the new origin JID since we've probably
					// just negotiated TLS and the client is comfortable telling us who it is
					// claiming to be now.
				case !origin.Equal(s.in.Info.From):
					return mask, nil, nState, fmt.Errorf("xmpp: stream origin %s does not match previously set origin %s", s.in.Info.From, origin)
				}
				switch {
				case location.Equal(jid.JID{}):
					// If we're a server receiving connection and "to" wasn't previously set,
					// just set it as this is the virtualhost we should use.
				case !location.Equal(s.in.Info.To):
					return mask, nil, nState, fmt.Errorf("xmpp: stream location %s does not match previously set location %s", s.in.Info.To, location)
				}

				location = in.To
				origin = in.From

				err = intstream.Send(s.Conn(), out, s.State()&S2S == S2S, websocket, stream.DefaultVersion, cfg.Lang, origin.String(), location.String(), attr.RandomID())
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.
				origin := s.LocalAddr()
				location := s.RemoteAddr()
				err = intstream.Send(s.Conn(), out, s.State()&S2S == S2S, websocket, stream.DefaultVersion, cfg.Lang, location.String(), origin.String(), "")
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}
				err = intstream.Expect(ctx, in, s.in.d, s.State()&Received == Received, websocket)
				if err != nil {
					nState.doRestart = false
					return mask, nil, nState, err
				}

				switch {
				case !location.Equal(s.in.Info.From):
					return mask, nil, nState, fmt.Errorf("xmpp: stream location %s does not match previously set location %s", s.in.Info.From, location)
				case !s.in.Info.To.Equal(jid.JID{}) && !origin.Equal(s.in.Info.To):
					// Technically this logic is not correct (we should only allow empty
					// "to" attributes if we didn't set "from" yet, so we should be
					// checking that). However, some servers don't send a "to" at all in
					// violation of the spec. See: https://issues.prosody.im/1625
					return mask, nil, nState, fmt.Errorf("xmpp: stream origin %s does not match previously set origin %s", s.in.Info.To, origin)
				}
			}
		}

		mask, rw, err = negotiateFeatures(ctx, s, data == nil, websocket, cfg.Features)
		nState.doRestart = rw != nil
		return mask, rw, nState, err
	}
}
