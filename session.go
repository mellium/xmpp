// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

// Errors returned by the XMPP package.
var (
	ErrInputStreamClosed  = errors.New("xmpp: attempted to read token from closed stream")
	ErrOutputStreamClosed = errors.New("xmpp: attempted to write token to closed stream")
)

var errNotStart = errors.New("xmpp: SendElement did not begin with a StartElement")

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
	conn net.Conn

	state SessionState

	origin   jid.JID
	location jid.JID

	// The stream feature namespaces advertised for the current streams.
	features map[string]interface{}

	// The negotiated features (by namespace) for the current session.
	negotiated map[string]struct{}

	sentIQs map[string]chan xmlstream.TokenReadCloser

	in struct {
		internal.StreamInfo
		d      xml.TokenReader
		ctx    context.Context
		cancel context.CancelFunc
	}
	out struct {
		internal.StreamInfo
		e tokenWriteFlusher
		sync.Mutex
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
func NegotiateSession(ctx context.Context, location, origin jid.JID, rw io.ReadWriter, received bool, negotiate Negotiator) (*Session, error) {
	if negotiate == nil {
		panic("xmpp: attempted to negotiate session with nil negotiator")
	}
	s := &Session{
		conn:       newConn(rw, nil),
		origin:     origin,
		location:   location,
		features:   make(map[string]interface{}),
		negotiated: make(map[string]struct{}),
		sentIQs:    make(map[string]chan xmlstream.TokenReadCloser),
	}
	if received {
		s.state |= Received
	}
	s.in.d = xml.NewDecoder(s.conn)
	s.out.e = xml.NewEncoder(s.conn)
	s.in.ctx, s.in.cancel = context.WithCancel(context.Background())

	// If rw was already a *tls.Conn, go ahead and mark the connection as secure
	// so that we don't try to negotiate StartTLS.
	if _, ok := s.conn.(*tls.Conn); ok {
		s.state |= Secure
	}

	// Call negotiate until the ready bit is set.
	var data interface{}
	for s.state&Ready == 0 {
		var mask SessionState
		var rw io.ReadWriter
		var err error
		mask, rw, data, err = negotiate(ctx, s, data)
		if err != nil {
			return s, err
		}
		if rw != nil {
			for k := range s.features {
				delete(s.features, k)
			}
			for k := range s.negotiated {
				delete(s.negotiated, k)
			}
			s.conn = newConn(rw, s.conn)
			s.in.d = xml.NewDecoder(s.conn)
			s.out.e = xml.NewEncoder(s.conn)
		}
		s.state |= mask
	}

	s.out.e = stanzaAddID(s.out.e)

	return s, nil
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
	return NegotiateSession(ctx, origin.Domain(), origin, conn, false, NewNegotiator(StreamConfig{Features: features}))
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
	return NegotiateSession(ctx, location, origin, conn, false, NewNegotiator(StreamConfig{S2S: true, Features: features}))
}

// NewClientSession attempts to use an existing connection (or any
// io.ReadWriter) to negotiate an XMPP client-to-server session.
// If the provided context is canceled before stream negotiation is complete an
// error is returned.
// After stream negotiation if the context is canceled it has no effect.
func NewClientSession(ctx context.Context, origin jid.JID, rw io.ReadWriter, received bool, features ...StreamFeature) (*Session, error) {
	return NegotiateSession(ctx, origin.Domain(), origin, rw, received, NewNegotiator(StreamConfig{
		Features: features,
	}))
}

// NewServerSession attempts to use an existing connection (or any
// io.ReadWriter) to negotiate an XMPP server-to-server session.
// If the provided context is canceled before stream negotiation is complete an
// error is returned.
// After stream negotiation if the context is canceled it has no effect.
func NewServerSession(ctx context.Context, location, origin jid.JID, rw io.ReadWriter, received bool, features ...StreamFeature) (*Session, error) {
	return NegotiateSession(ctx, location, origin, rw, received, NewNegotiator(StreamConfig{
		S2S:      true,
		Features: features,
	}))
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
// If serve handles an incoming IQ stanza and the handler does not write a
// response (an IQ with the same ID and type "result" or "error"), Serve writes
// an error IQ with a service-unavailable payload.
//
// If the user closes the output stream by calling Close, Serve continues until
// the input stream is closed by the remote entity as above, or the deadline set
// by SetCloseDeadline is reached in which case a timeout error is returned.
func (s *Session) Serve(h Handler) error {
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

type nopHandler struct{}

func (nopHandler) HandleXMPP(_ xmlstream.TokenReadWriter, _ *xml.StartElement) error {
	return nil
}

type iqResponder struct {
	r xml.TokenReader
	c chan xmlstream.TokenReadCloser
}

func (r iqResponder) Token() (xml.Token, error) {
	return r.r.Token()
}

func (r iqResponder) Close() error {
	close(r.c)
	return nil
}

func (s *Session) handleInputStream(handler Handler) (err error) {
	if handler == nil {
		handler = nopHandler{}
	}

	defer func() {
		s.closeInputStream()
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
		if err != nil {
			// If this was a read timeout, don't try to send it. Just try to read
			// again.
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
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

		// If this is a stanza, normalize the "from" attribute.
		if isStanza(start.Name) {
			for i, attr := range start.Attr {
				if attr.Name.Local == "from" /*&& attr.Name.Space == start.Name.Space*/ {
					local := s.LocalAddr().Bare().String()
					// Try a direct comparison first to avoid expensive JID parsing.
					// TODO: really we should be parsing the JID here in case the server
					// is using a different version of PRECIS, stringprep, etc. and the
					// canonical representation isn't the same.
					if attr.Value == local {
						start.Attr[i].Value = ""
					}
					break
				}
			}
		}

		var id string
		var needsResp bool
		if isIQ(start.Name) {
			id = getID(start)

			// If this is a response IQ (ie. an "error" or "result") check if we're
			// handling it as part of a SendElement call.
			// If not, record this so that we can check if the user sends a response
			// later.
			if !iqNeedsResp(start.Attr) {
				c := s.sentIQs[id]
				if c == nil {
					goto noreply
				}

				c <- iqResponder{
					r: xmlstream.MultiReader(xmlstream.Token(start), xmlstream.Inner(s), xmlstream.Token(start.End())),
					c: c,
				}
				<-c
				// Consume the rest of the stream before continuing the loop.
				_, err = xmlstream.Copy(discard, s)
				if err != nil {
					return s.sendError(err)
				}
				continue
			} else {
				needsResp = true
			}
		}

	noreply:

		rw := &responseChecker{
			TokenReader: xmlstream.Inner(s),
			TokenWriter: s,
			id:          id,
		}
		if err = handler.HandleXMPP(rw, &start); err != nil {
			return s.sendError(err)
		}

		// If the user did not write a response to an IQ, send a default one.
		if needsResp && !rw.wroteResp {
			_, err := xmlstream.Copy(s, stanza.WrapIQ(stanza.IQ{
				ID:   id,
				Type: stanza.ErrorIQ,
			}, stanza.Error{
				Type:      stanza.Cancel,
				Condition: stanza.ServiceUnavailable,
			}.TokenReader()))
			if err != nil {
				return err
			}
		}

		if err := s.Flush(); err != nil {
			return err
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

type responseChecker struct {
	xml.TokenReader
	xmlstream.TokenWriter
	id        string
	wroteResp bool
	level     int
}

func (rw *responseChecker) EncodeToken(t xml.Token) error {
	switch tok := t.(type) {
	case xml.StartElement:
		id := getID(tok)
		if rw.level < 1 && isIQ(tok.Name) && id == rw.id && !iqNeedsResp(tok.Attr) {
			rw.wroteResp = true
		}
		rw.level++
	case xml.EndElement:
		rw.level--
	}
	return rw.TokenWriter.EncodeToken(t)
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
func (s *Session) Conn() net.Conn {
	return s.conn
}

// Token satisfies the xml.TokenReader interface for Session.
func (s *Session) Token() (xml.Token, error) {
	if s.state&InputStreamClosed == InputStreamClosed {
		return nil, ErrInputStreamClosed
	}
	return s.in.d.Token()
}

// EncodeToken satisfies the xmlstream.TokenWriter interface.
func (s *Session) EncodeToken(t xml.Token) error {
	if s.state&OutputStreamClosed == OutputStreamClosed {
		return ErrOutputStreamClosed
	}
	return s.out.e.EncodeToken(t)
}

// Flush satisfies the xmlstream.TokenWriter interface.
func (s *Session) Flush() error {
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
	return s.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (s *Session) LocalAddr() jid.JID {
	if (s.state & Received) == Received {
		return s.location
	}
	return s.origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (s *Session) RemoteAddr() jid.JID {
	if (s.state & Received) == Received {
		return s.origin
	}
	return s.location
}

// SetCloseDeadline sets a deadline for the input stream to be closed by the
// other side.
// If the input stream is not closed by the deadline, the input stream is marked
// as closed and any blocking calls to Serve will return an error.
// This is normally called just before a call to Close.
func (s *Session) SetCloseDeadline(t time.Time) error {
	oldCancel := s.in.cancel
	s.in.ctx, s.in.cancel = context.WithDeadline(context.Background(), t)
	if oldCancel != nil {
		oldCancel()
	}
	return s.Conn().SetReadDeadline(t)
}

// Send transmits the first element read from the provided token reader.
//
// For more information, see SendElement.
func (s *Session) Send(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error) {
	return s.SendElement(ctx, r, xml.StartElement{})
}

func iqNeedsResp(attrs []xml.Attr) bool {
	var typ string
	for _, attr := range attrs {
		if attr.Name.Local == "type" {
			typ = attr.Value
			break
		}
	}

	return typ == "get" || typ == "set"
}

func isIQ(name xml.Name) bool {
	return name.Local == "iq" && (name.Space == "" || name.Space == ns.Client || name.Space == ns.Server)
}

func isStanza(name xml.Name) bool {
	return (name.Local == "iq" || name.Local == "message" || name.Local == "presence") &&
		(name.Space == "" || name.Space == ns.Client || name.Space == ns.Server)
}

func getID(start xml.StartElement) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == "id" {
			return attr.Value
		}
	}
	return ""
}

// SendElement transmits the first element read from the provided token reader
// using start as the outermost tag in the encoding.
//
// If the element is an info/query (IQ) stanza, Send blocks until a response is
// received and then returns a reader from which the response can be read.
// If the input stream is not being processed (a call to Serve is not running),
// SendElement may block forever.
// If the provided context is closed before the response is received SendElement
// immediately returns an error and any response received at a later time will
// not be associated with the original request.
// The response does not need to be consumed in its entirety, but it must be
// closed before stream processing will resume.
// If an error is returned, xml.TokenReader will be nil; the converse is not
// necessarily true.
//
// SendElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendElement(ctx context.Context, r xml.TokenReader, start xml.StartElement) (xmlstream.TokenReadCloser, error) {
	s.out.Lock()
	defer s.out.Unlock()

	if start.Name.Local == "" {
		tok, err := r.Token()
		if err != nil {
			return nil, err
		}

		var ok bool
		start, ok = tok.(xml.StartElement)
		if !ok {
			return nil, errNotStart
		}
	}

	// If this is not an IQ (or is an IQ that's not of type "set" or "get") we
	// don't expect a response and merely transmit the information.
	if !isIQ(start.Name) || !iqNeedsResp(start.Attr) {
		err := s.EncodeToken(start)
		if err != nil {
			return nil, err
		}
		_, err = xmlstream.Copy(s, xmlstream.Inner(r))
		if err != nil {
			return nil, err
		}
		err = s.EncodeToken(start.End())
		if err != nil {
			return nil, err
		}
		return nil, s.Flush()
	}

	// We need to add an id to the IQ if one wasn't already set by the user so
	// that we can use it to associate the response with the original query.
	id := getID(start)
	if id == "" {
		id = internal.RandomID()
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: id})
	}

	c := make(chan xmlstream.TokenReadCloser)
	s.sentIQs[id] = c

	err := s.EncodeToken(start)
	if err != nil {
		return nil, err
	}
	_, err = xmlstream.Copy(s, xmlstream.Inner(r))
	if err != nil {
		return nil, err
	}
	err = s.EncodeToken(start.End())
	if err != nil {
		return nil, err
	}
	err = s.Flush()
	if err != nil {
		return nil, err
	}

	select {
	case rr := <-c:
		delete(s.sentIQs, id)
		return rr, nil
	case <-ctx.Done():
		delete(s.sentIQs, id)
		close(c)
		return nil, ctx.Err()
	}
}

// closeInputStream immediately marks the input stream as closed and cancels any
// deadlines associated with it.
func (s *Session) closeInputStream() {
	s.state |= InputStreamClosed
	s.in.cancel()
}

type wrapWriter struct {
	encode func(t xml.Token) error
	flush  func() error
}

func (w wrapWriter) EncodeToken(t xml.Token) error { return w.encode(t) }
func (w wrapWriter) Flush() error                  { return w.flush() }

type tokenWriteFlusher interface {
	xmlstream.TokenWriter
	xmlstream.Flusher
}

func stanzaAddID(w tokenWriteFlusher) tokenWriteFlusher {
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
