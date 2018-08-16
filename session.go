// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// Errors returned by the XMPP package.
var (
	ErrInputStreamClosed  = errors.New("xmpp: attempted to read token from closed stream")
	ErrOutputStreamClosed = errors.New("xmpp: attempted to write token to closed stream")
)

// SessionState is a bitmask that represents the current state of an XMPP
// session. For a description of each bit, see the various SessionState typed
// constants.
type SessionState uint8

const (
	// Secure indicates that the underlying connection has been secured. For
	// instance, after STARTTLS has been performed or if a pre-secured connection
	// is being used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Authn indicates that the session has been authenticated (probably with
	// SASL).
	Authn

	// Ready indicates that the session is fully negotiated and that XMPP stanzas
	// may be sent and received.
	Ready

	// Received indicates that the session was initiated by a foreign entity.
	Received

	// OutputStreamClosed indicates that the output stream has been closed with a
	// stream end tag.  When set all write operations will return an error even if
	// the underlying TCP connection is still open.
	OutputStreamClosed

	// InputStreamClosed indicates that the input stream has been closed with a
	// stream end tag. When set all read operations will return an error.
	InputStreamClosed
)

// A Session represents an XMPP session comprising an input and an output XML
// stream.
type Session struct {
	conn *Conn

	state SessionState
	slock sync.RWMutex

	origin   jid.JID
	location jid.JID

	// The stream feature namespaces advertised for the current streams.
	features map[string]interface{}

	// The negotiated features (by namespace) for the current session.
	negotiated map[string]struct{}

	in struct {
		sync.Mutex
		internal.StreamInfo
		d      xml.TokenReader
		ctx    context.Context
		cancel context.CancelFunc
	}
	out struct {
		internal.StreamInfo
		e xmlstream.TokenWriter
	}
}

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

// NegotiateSession creates an XMPP session using a custom negotiate function.
// Calling NegotiateSession with a nil Negotiator panics.
//
// For more information see the Negotiator type.
func NegotiateSession(ctx context.Context, location, origin jid.JID, rw io.ReadWriter, negotiate Negotiator) (*Session, error) {
	if negotiate == nil {
		panic("xmpp: attempted to negotiate session with nil negotiator")
	}
	s := &Session{
		conn:       newConn(rw),
		origin:     origin,
		location:   location,
		features:   make(map[string]interface{}),
		negotiated: make(map[string]struct{}),
	}
	s.in.d = xml.NewDecoder(s.conn)
	s.out.e = xml.NewEncoder(s.conn)
	s.in.ctx, s.in.cancel = context.WithCancel(context.Background())

	// If rw was already a *tls.Conn or some other Conn that is secure, go ahead
	// and mark the connection as secure so that we don't try to negotiate
	// StartTLS.
	if s.conn.Secure() {
		s.state |= Secure
	}

	// Call negotiate until the ready bit is set.
	var data interface{} = true
	for s.state&Ready == 0 {
		var mask SessionState
		var rw io.ReadWriter
		var err error
		mask, rw, data, err = negotiate(ctx, s, data)
		if err != nil {
			return s, err
		}
		if rw != nil {
			s.features = make(map[string]interface{})
			s.negotiated = make(map[string]struct{})
			s.conn = newConn(rw)
			s.in.d = xml.NewDecoder(s.conn)
			s.out.e = xml.NewEncoder(s.conn)
		}
		s.state |= mask
	}

	s.out.e = stanzaAddID(s.out.e)

	return s, nil
}

type wrapWriter struct {
	encode func(t xml.Token) error
	flush  func() error
}

func (w wrapWriter) EncodeToken(t xml.Token) error { return w.encode(t) }
func (w wrapWriter) Flush() error                  { return w.flush() }

func stanzaAddID(w xmlstream.TokenWriter) xmlstream.TokenWriter {
	depth := 0
	return wrapWriter{
		encode: func(t xml.Token) error {
		tokswitch:
			switch tok := t.(type) {
			case xml.StartElement:
				depth++
				if depth == 1 && tok.Name.Local == "iq" {
					for _, attr := range tok.Attr {
						if attr.Name.Local == "id" {
							break tokswitch
						}
					}
					tok.Attr = append(tok.Attr, xml.Attr{
						Name:  xml.Name{Local: "id"},
						Value: internal.RandomID(),
					})
					t = tok
				}
			case xml.EndElement:
				depth--
			}
			return w.EncodeToken(t)
		},
		flush: w.Flush,
	}
}

// DialClientSession uses a default dialer to create a TCP connection and
// attempts to negotiate an XMPP session over it.
//
// If the provided context is canceled after stream negotiation is complete it
// has no effect on the session.
func DialClientSession(ctx context.Context, origin jid.JID, features ...StreamFeature) (*Session, error) {
	conn, err := DialClient(ctx, "tcp", origin)
	if err != nil {
		return nil, err
	}
	return NegotiateSession(ctx, origin.Domain(), origin, conn, NewNegotiator(StreamConfig{Features: features}))
}

// DialServerSession uses a default dialer to create a TCP connection and
// attempts to negotiate an XMPP session over it.
//
// If the provided context is canceled after stream negotiation is complete it
// has no effect on the session.
func DialServerSession(ctx context.Context, location, origin jid.JID, features ...StreamFeature) (*Session, error) {
	conn, err := DialServer(ctx, "tcp", location)
	if err != nil {
		return nil, err
	}
	return NegotiateSession(ctx, location, origin, conn, NewNegotiator(StreamConfig{S2S: true, Features: features}))
}

// NewClientSession attempts to use an existing connection (or any
// io.ReadWriter) to negotiate an XMPP client-to-server session.
// If the provided context is canceled before stream negotiation is complete an
// error is returned.
// After stream negotiation if the context is canceled it has no effect.
func NewClientSession(ctx context.Context, origin jid.JID, rw io.ReadWriter, features ...StreamFeature) (*Session, error) {
	return NegotiateSession(ctx, origin.Domain(), origin, rw, NewNegotiator(StreamConfig{Features: features}))
}

// NewServerSession attempts to use an existing connection (or any
// io.ReadWriter) to negotiate an XMPP server-to-server session.
// If the provided context is canceled before stream negotiation is complete an
// error is returned.
// After stream negotiation if the context is canceled it has no effect.
func NewServerSession(ctx context.Context, location, origin jid.JID, rw io.ReadWriter, features ...StreamFeature) (*Session, error) {
	return NegotiateSession(ctx, location, origin, rw, NewNegotiator(StreamConfig{S2S: true, Features: features}))
}

// Serve decodes incoming XML tokens from the connection and delegates handling
// them to h.
// If an error is returned from the handler and it is of type stanza.Error or
// stream.Error, the error is marshaled and sent over the XML stream.
// If any other error type is returned, it is marshaled as an
// undefined-condition StreamError.
// If a stream error is received while serving it is not passed to the handler.
// Instead, Serve unmarshals the error, closes the session, and returns it (h
// handles stanza level errors, the session handles stream level errors).
//
// If Serve is called concurrently the second invocation blocks until the first
// returns.
// If the input stream is closed, Serve returns.
// Serve does not close the output stream.
func (s *Session) Serve(h Handler) error {
	s.in.Lock()
	defer s.in.Unlock()

	return s.handleInputStream(h)
}

// sendError transmits an error on the session. If the error is not a standard
// stream error an UndefinedCondition stream error is sent.
// If an error is returned (the original error or a different one), it has not
// been handled fully and must be handled by the caller.
func (s *Session) sendError(err error) (e error) {
	switch typErr := err.(type) {
	case stream.Error:
		if _, e = typErr.WriteXML(s); e != nil {
			return e
		}
		if e = s.Close(); e != nil {
			return e
		}
		return err
	}
	// TODO: What should we do here? RFC 6120 §4.9.3.21. undefined-condition
	// says:
	//
	//     The error condition is not one of those defined by the other
	//     conditions in this list; this error condition SHOULD NOT be used
	//     except in conjunction with an application-specific condition.
	if _, e = stream.UndefinedCondition.WriteXML(s); e != nil {
		return e
	}
	return err
}

func (s *Session) handleInputStream(handler Handler) (err error) {
	defer func() {
		e := s.Close()
		if err == nil {
			err = e
		}
	}()

	discard := xmlstream.Discard()

	for {
		select {
		case <-s.in.ctx.Done():
			return s.in.ctx.Err()
		default:
		}
		tok, err := s.Token()
		// TODO: If this is a network issue we should return it, if not we should
		// handle it.
		if err != nil {
			return s.sendError(err)
		}

		var start xml.StartElement
		switch t := tok.(type) {
		case xml.StartElement:
			start = t
		case xml.EndElement:
			if t.Name.Space == ns.Stream && t.Name.Local == "stream" {
				return nil
			}
			// If this is a stream level end element but not </stream:stream>,
			// something is really weird…
			return s.sendError(stream.BadFormat)
		default:
			// If this isn't a start element, the stream is in a bad state.
			return s.sendError(stream.BadFormat)
		}

		rw := struct {
			xml.TokenReader
			xmlstream.TokenWriter
		}{
			TokenReader: xmlstream.Inner(s),
			TokenWriter: s,
		}

		// Handle stream errors and unknown stream namespaced tokens first, before
		// delegating to the normal handler.
		if start.Name.Space == ns.Stream {
			switch start.Name.Local {
			case "error":
				// TODO: Unmarshal the error and return it.
				return nil
			default:
				return s.sendError(stream.UnsupportedStanzaType)
			}
		}

		if err = handler.HandleXMPP(rw, &start); err != nil {
			return s.sendError(err)
		}
		// Advance to the end of the current element before attempting to read the
		// next.
		//
		// TODO: Error handling should be the same here as it would be for the rest
		// of this loop.
		_, err = xmlstream.Copy(discard, rw)
		if err != nil {
			return s.sendError(err)
		}
	}
}

// Feature checks if a feature with the given namespace was advertised
// by the server for the current stream. If it was data will be the canonical
// representation of the feature as returned by the feature's Parse function.
func (s *Session) Feature(namespace string) (data interface{}, ok bool) {
	// TODO: Make the features struct actually store the parsed representation.
	data, ok = s.features[namespace]
	return data, ok
}

// Conn returns the Session's backing connection.
//
// This should almost never be read from or written to, but is useful during
// stream negotiation for wrapping the existing connection in a new layer (eg.
// compression or TLS).
func (s *Session) Conn() *Conn {
	return s.conn
}

// Token satisfies the xml.TokenReader interface for Session.
func (s *Session) Token() (xml.Token, error) {
	s.slock.RLock()
	defer s.slock.RUnlock()

	if s.state&InputStreamClosed == InputStreamClosed {
		return nil, ErrInputStreamClosed
	}
	return s.in.d.Token()
}

// EncodeToken satisfies the xmlstream.TokenWriter interface.
func (s *Session) EncodeToken(t xml.Token) error {
	s.slock.RLock()
	defer s.slock.RUnlock()

	if s.state&OutputStreamClosed == OutputStreamClosed {
		return ErrOutputStreamClosed
	}
	return s.out.e.EncodeToken(t)
}

// Flush satisfies the xmlstream.TokenWriter interface.
func (s *Session) Flush() error {
	s.slock.RLock()
	defer s.slock.RUnlock()

	if s.state&OutputStreamClosed == OutputStreamClosed {
		return ErrOutputStreamClosed
	}
	return s.out.e.Flush()
}

// Close ends the output stream (by sending a closing </stream:stream> token).
// It does not close the underlying connection.
// Calling Close() multiple times will only result in one closing
// </stream:stream> being sent.
func (s *Session) Close() error {
	s.slock.Lock()
	defer s.slock.Unlock()
	if s.state&OutputStreamClosed == OutputStreamClosed {
		return nil
	}

	s.state |= OutputStreamClosed
	// We wrote the opening stream instead of encoding it, so do the same with the
	// closing to ensure that the encoder doesn't think the tokens are mismatched.
	_, err := s.Conn().Write([]byte(`</stream:stream>`))
	return err
}

// State returns the current state of the session. For more information, see the
// SessionState type.
func (s *Session) State() SessionState {
	s.slock.RLock()
	defer s.slock.RUnlock()
	return s.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (s *Session) LocalAddr() jid.JID {
	s.slock.RLock()
	defer s.slock.RUnlock()
	if (s.state & Received) == Received {
		return s.location
	}
	return s.origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (s *Session) RemoteAddr() jid.JID {
	s.slock.RLock()
	defer s.slock.RUnlock()
	if (s.state & Received) == Received {
		return s.origin
	}
	return s.location
}

// StreamConfig contains options for configuring the default Negotiator.
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// S2S causes the server to attempt to dial a server-to-server connection.
	S2S bool

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature
}

// NewNegotiator creates a Negotiator that uses a collection of
// StreamFeatures to negotiate an XMPP server-to-server session.
func NewNegotiator(cfg StreamConfig) Negotiator {
	return negotiator(cfg.S2S, cfg.Lang, cfg.Features)
}

func negotiator(s2s bool, lang string, features []StreamFeature) Negotiator {
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
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), s2s, internal.DefaultVersion, lang, s.location.String(), s.origin.String(), internal.RandomID())
				if err != nil {
					return mask, nil, false, err
				}
			} else {
				// If we're the initiating entity, send a new stream and then wait for
				// one in response.
				s.out.StreamInfo, err = internal.SendNewStream(s.Conn(), s2s, internal.DefaultVersion, lang, s.location.String(), s.origin.String(), "")
				if err != nil {
					return mask, nil, false, err
				}
				s.in.StreamInfo, err = internal.ExpectNewStream(ctx, s.in.d, s.State()&Received == Received)
				if err != nil {
					return mask, nil, false, err
				}
			}
		}

		mask, rw, err = negotiateFeatures(ctx, s, features)
		return mask, rw, rw != nil, err
	}
}
