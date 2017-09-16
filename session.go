// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"io"
	"net"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
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
	config *Config

	// If the initial ReadWriter is a conn, save a reference to that as well so
	// that we can use it directly without type casting constantly.
	conn net.Conn
	rw   io.ReadWriter

	state SessionState

	// The actual origin of this conn (we don't want to mutate the config, so if
	// this origin exists and is different from the one in config, eg. because the
	// server did not assign us the resourcepart we requested, this is canonical).
	origin *jid.JID

	// The stream feature namespaces advertised for the current streams.
	features map[string]interface{}
	flock    sync.Mutex

	// The negotiated features (by namespace) for the current session.
	negotiated map[string]struct{}

	in struct {
		sync.Mutex
		streamInfo
		d      xmlstream.TokenReader
		ctx    context.Context
		cancel context.CancelFunc
	}
	out struct {
		sync.Mutex
		streamInfo
		e *xml.Encoder
	}
}

// NewSession attempts to use an existing connection (or any io.ReadWriter) to
// negotiate an XMPP session based on the given config. If the provided context
// is canceled before stream negotiation is complete an error is returned. After
// stream negotiation if the context is canceled it has no effect.
func NewSession(ctx context.Context, config *Config, rw io.ReadWriter) (*Session, error) {
	s := &Session{
		config: config,
		rw:     rw,
	}
	s.in.ctx, s.in.cancel = context.WithCancel(context.Background())

	if conn, ok := rw.(net.Conn); ok {
		s.conn = conn
	}

	err := s.negotiateStreams(ctx, rw)
	if err != nil {
		return nil, err
	}
	return s, err
}

// Serve decodes incoming XML tokens from the connection and delegates handling
// them to the provided handler.
// If an error is returned from the handler and it is of type stanza.Error or
// stream.Error, the error is marshaled and sent over the XML stream. If any
// other error type is returned, it is marshaled as an undefined-condition
// StreamError. If a stream error is received while serving it is not passed to
// the handler. Instead, Serve unmarshals the error, closes the session, and
// returns it (handlers handle stanza level errors, the session handles stream
// level errors).
func (s *Session) Serve(handler Handler) error {
	return s.handleInputStream(handler)
}

// Feature checks if a feature with the given namespace was advertised
// by the server for the current stream. If it was data will be the canonical
// representation of the feature as returned by the feature's Parse function.
func (s *Session) Feature(namespace string) (data interface{}, ok bool) {
	s.flock.Lock()
	defer s.flock.Unlock()

	// TODO: Make the features struct actually store the parsed representation.
	data, ok = s.features[namespace]
	return
}

// Conn returns the Session's backing net.Conn or other io.ReadWriter.
func (s *Session) Conn() io.ReadWriter {
	return s.rw
}

// TokenReader returns the XML token reader that was used to negotiate the
// latest stream.
func (s *Session) TokenReader() xmlstream.TokenReader {
	return s.in.d
}

// Encoder returns the XML encoder that was used to negotiate the latest stream.
func (s *Session) Encoder() *xml.Encoder {
	return s.out.e
}

// Config returns the connections config.
func (s *Session) Config() *Config {
	return s.config
}

// Close ends the output stream (by sending a closing </stream:stream> token).
// It does not close the underlying connection.
// Calling Close() multiple times will only result in one closing
// </stream:stream> being sent.
func (s *Session) Close() (err error) {
	s.out.Lock()
	defer s.out.Unlock()
	if s.state&OutputStreamClosed != OutputStreamClosed {
		s.Encoder().EncodeToken(xml.EndElement{
			Name: xml.Name{Local: "stream:stream"},
		})
	}

	return
}

// State returns the current state of the session. For more information, see the
// SessionState type.
func (s *Session) State() SessionState {
	return s.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (s *Session) LocalAddr() *jid.JID {
	if (s.state & Received) == Received {
		return s.config.Location
	}
	if s.origin != nil {
		return s.origin
	}
	return s.config.Origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (s *Session) RemoteAddr() *jid.JID {
	if (s.state & Received) == Received {
		return s.config.Origin
	}
	return s.config.Location
}

func (s *Session) handleInputStream(handler Handler) error {
	s.in.Lock()
	defer s.in.Unlock()
	defer s.Close()

	for {
		select {
		case <-s.in.ctx.Done():
			return nil
		default:
		}
		tok, err := s.TokenReader().Token()
		if err != nil {
			select {
			case <-s.in.ctx.Done():
				return nil
			default:
				// TODO: We need a way to figure out if this was an XML error or an
				// error with the underlying connection.
				return s.Encoder().Encode(stream.BadFormat)
			}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "error" && t.Name.Space == ns.Stream {
				e := stream.Error{}
				err = xml.NewTokenDecoder(s.TokenReader()).DecodeElement(&e, &t)
				if err != nil {
					return err
				}
				return e
			}
			if err = handler.HandleXMPP(s, &t); err != nil {
				switch err.(type) {
				case stanza.Error:
					err = s.Encoder().Encode(err)
					if err != nil {
						return err
					}
				case stream.Error:
					return s.Encoder().Encode(err)
				default:
					// TODO: Should this error have a payload?
					return s.Encoder().Encode(stream.UndefinedCondition)
				}
			}
		case xml.EndElement:
			if t.Name.Space == ns.Stream && t.Name.Local == "stream" {
				s.state |= InputStreamClosed
				return nil
			}
		default:
			select {
			case <-s.in.ctx.Done():
				return nil
			default:
				// TODO: We need a way to figure out if this was an XML error or an
				// error with the underlying connection.
				return s.Encoder().Encode(stream.BadFormat)
			}
		}
	}
}
