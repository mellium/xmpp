// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmpp/jid"
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

// A Session represents an XMPP connection that can perform SRV lookups for a given
// server and connect to the correct ports.
type Session struct {
	config *Config
	rwc    io.ReadWriteCloser

	// If the initial rwc is a conn, save a reference to that as well so that we
	// can set deadlines on it later even if the rwc is upgraded.
	conn net.Conn

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
		stream
		d *xml.Decoder
	}
	out struct {
		sync.Mutex
		stream
		e *xml.Encoder
	}
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

// NewSession attempts to use an existing connection (or any io.ReadWriteCloser) to
// negotiate an XMPP session based on the given config. If the provided context
// is canceled before stream negotiation is complete an error is returned. After
// stream negotiation if the context is canceled it has no effect.
func NewSession(ctx context.Context, config *Config, rwc io.ReadWriteCloser) (*Session, error) {
	s := &Session{
		config: config,
	}

	if conn, ok := rwc.(net.Conn); ok {
		s.conn = conn
	}

	return s, s.negotiateStreams(ctx, rwc)
}

// Conn returns the Session's backing net.Conn or other ReadWriteCloser.
func (s *Session) Conn() io.ReadWriteCloser {
	return s.rwc
}

// Decoder returns the XML decoder that was used to negotiate the latest stream.
func (s *Session) Decoder() *xml.Decoder {
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

func (s *Session) read(b []byte) (n int, err error) {
	s.in.Lock()
	defer s.in.Unlock()

	if s.state&InputStreamClosed == InputStreamClosed {
		return 0, errors.New("XML input stream is closed")
	}

	return s.rwc.Read(b)
}

func (s *Session) write(b []byte) (n int, err error) {
	s.out.Lock()
	defer s.out.Unlock()

	if s.state&OutputStreamClosed == OutputStreamClosed {
		return 0, errors.New("XML output stream is closed")
	}

	return s.rwc.Write(b)
}

// Close closes the underlying connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (s *Session) Close() error {
	return s.rwc.Close()
}

// State returns the current state of the session. For more information, see the
// SessionState type.
func (s *Session) State() SessionState {
	return s.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (s *Session) LocalAddr() net.Addr {
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
func (s *Session) RemoteAddr() net.Addr {
	if (s.state & Received) == Received {
		return s.config.Origin
	}
	return s.config.Location
}

var errSetDeadline = errors.New("xmpp: cannot set deadline: not using a net.Conn")

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations fail with a timeout
// (see type Error) instead of blocking. The deadline applies to all future I/O,
// not just the immediately following call to Read or Write.
//
// An idle timeout can be implemented by repeatedly extending the deadline after
// successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (s *Session) SetDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetDeadline(t)
	}
	return errSetDeadline
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (s *Session) SetReadDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetReadDeadline(t)
	}
	return errSetDeadline
}

// SetWriteDeadline sets the deadline for future Write calls. Even if write
// times out, it may return n > 0, indicating that some of the data was
// successfully written. A zero value for t means Write will not time out.
func (s *Session) SetWriteDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetWriteDeadline(t)
	}
	return errSetDeadline
}
