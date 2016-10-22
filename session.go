// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"

	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/streamerror"
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
		stream
		d      *xml.Decoder
		ctx    context.Context
		cancel context.CancelFunc
	}
	out struct {
		sync.Mutex
		stream
		e *xml.Encoder
	}
}

// Send is used to marshal an element into XML and transmit it over the egress
// stream. If the interface is composed of a stanza (IQ, Message, or Presence)
// and the from or id attributes are empty, an appropriate value is inserted in
// the output XML.
func (s *Session) Send(v interface{}) error {

	// TODO: This is horrifying, and probably buggy. There has got to be a better
	//       way that I haven't thought of…

	switch copier := v.(type) {
	case interface {
		copyIQ() IQ
	}:
		iq := copier.copyIQ()
		iq.ID = internal.RandomID()

		val, err := getCopy(reflect.ValueOf(v))
		if err != nil {
			return nil
		}
		val.FieldByName("IQ").Set(reflect.ValueOf(iq))
		v = val.Interface()
	}

	return s.out.e.Encode(v)
}

func getCopy(val reflect.Value) (v reflect.Value, err error) {
	for val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return v, fmt.Errorf("Failed")
		}
		val = val.Elem()
	}
	return reflect.New(val.Type()).Elem(), nil
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
	if config.Handler != nil {
		go s.handleInputStream()
	}
	return s, err
}

func (s *Session) handleInputStream() {
	for {
		select {
		case <-s.in.ctx.Done():
			return
		default:
		}
		// TODO: This needs to be cancelable somehow.
		tok, err := s.Decoder().Token()
		if err != nil {
			select {
			case <-s.in.ctx.Done():
				return
			default:
				// TODO: We need a way to figure out if this was an XML error or an error
				// with the underlying connection.
				s.Encoder().Encode(streamerror.BadFormat)
			}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			s.config.Handler.HandleXMPP(s, &t)
		default:
			select {
			case <-s.in.ctx.Done():
				return
			default:
				// TODO: We need a way to figure out if this was an XML error or an error
				// with the underlying connection.
				s.Encoder().Encode(streamerror.BadFormat)
			}
		}
	}
}

// Conn returns the Session's backing net.Conn or other ReadWriter.
func (s *Session) Conn() io.ReadWriter {
	return s.rw
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

	n, err = s.rw.Read(b)
	return
}

func (s *Session) write(b []byte) (n int, err error) {
	s.out.Lock()
	defer s.out.Unlock()

	if s.state&OutputStreamClosed == OutputStreamClosed {
		return 0, errors.New("XML output stream is closed")
	}

	n, err = s.rw.Write(b)
	return
}

// Close ends the output stream and blocks until the remote client closes the
// input stream.
func (s *Session) Close() (err error) {
	// TODO: Block until input stream is closed?
	_, err = s.write([]byte(`</stream:stream>`))
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
