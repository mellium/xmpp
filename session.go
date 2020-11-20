// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/internal/ns"
	intstream "mellium.im/xmpp/internal/stream"
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

const closeStreamTag string = `</stream:stream>`

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
	conn      net.Conn
	connState func() tls.ConnectionState

	state      SessionState
	stateMutex sync.RWMutex

	origin   jid.JID
	location jid.JID

	// The stream feature namespaces advertised for the current streams.
	features map[string]interface{}

	// The negotiated features (by namespace) for the current session.
	negotiated map[string]struct{}

	sentIQMutex sync.Mutex
	sentIQs     map[string]chan xmlstream.TokenReadCloser

	in struct {
		intstream.Info
		d      xml.TokenReader
		ctx    context.Context
		cancel context.CancelFunc
		sync.Locker
	}
	out struct {
		intstream.Info
		e tokenWriteFlusher
		sync.Locker
	}
}

var _ tlsConn = (*Session)(nil)

// ConnectionState returns the underlying connections TLS state or the zero
// value if TLS has not been negotiated.
func (s *Session) ConnectionState() tls.ConnectionState {
	if s.connState == nil {
		return tls.ConnectionState{}
	}
	return s.connState()
}

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
	if tc, ok := s.conn.(tlsConn); ok {
		s.connState = tc.ConnectionState
	}
	if received {
		// We don't need to lock the state mutex yet since we haven't returned
		// session and nothing else can access it.
		s.state |= Received
	}
	s.out.Locker = &sync.Mutex{}
	s.in.Locker = &sync.Mutex{}
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
			if tc, ok := s.conn.(tlsConn); ok {
				s.connState = tc.ConnectionState
			}
			s.in.d = xml.NewDecoder(s.conn)
			s.out.e = xml.NewEncoder(s.conn)
		}
		s.state |= mask
	}

	s.in.d = intstream.Reader(s.in.d)
	s.out.e = stanzaAddID(s.out.e)

	return s, nil
}

// DialClientSession uses a default dialer to create a TCP connection and
// attempts to negotiate an XMPP session over it.
//
// If the provided context is canceled after stream negotiation is complete it
// has no effect on the session.
func DialClientSession(ctx context.Context, origin jid.JID, features ...StreamFeature) (*Session, error) {
	conn, err := dial.Client(ctx, "tcp", origin)
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
	conn, err := dial.Server(ctx, "tcp", location)
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
// Serve takes a lock on the input and output stream before calling the handler,
// so the handler should not close over the session or use any of its send
// methods or a deadlock will occur.
// After Serve finishes running the handler, it flushes the output stream.
func (s *Session) Serve(h Handler) (err error) {
	if h == nil {
		h = nopHandler{}
	}

	defer func() {
		s.closeInputStream()
		e := s.Close()
		if err == nil {
			err = e
		}
	}()

	for {
		select {
		case <-s.in.ctx.Done():
			return s.in.ctx.Err()
		default:
		}
		err := handleInputStream(s, h)
		switch err {
		case nil:
			// No error and no sentinal error telling us to shut down; try again!
		case io.EOF:
			return nil
		default:
			return s.sendError(err)
		}
	}
}

// sendError transmits an error on the session. If the error is not a standard
// stream error an UndefinedCondition stream error is sent.
// If an error is returned (the original error or a different one), it has not
// been handled fully and must be handled by the caller.
func (s *Session) sendError(err error) (e error) {
	s.out.Lock()
	defer s.out.Unlock()
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	if s.state&OutputStreamClosed == OutputStreamClosed {
		return err
	}

	switch typErr := err.(type) {
	case stream.Error:
		if _, e = typErr.WriteXML(s.out.e); e != nil {
			return e
		}
		if e = s.closeSession(); e != nil {
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
	if _, e = stream.UndefinedCondition.WriteXML(s.out.e); e != nil {
		return e
	}
	return err
}

type nopHandler struct{}

func (nopHandler) HandleXMPP(xmlstream.TokenReadEncoder, *xml.StartElement) error {
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

func handleInputStream(s *Session, handler Handler) (err error) {
	discard := xmlstream.Discard()
	rc := s.TokenReader()
	defer rc.Close()
	r := intstream.Reader(rc)

	tok, err := r.Token()
	if err != nil {
		return err
	}

	var start xml.StartElement
	switch t := tok.(type) {
	case xml.StartElement:
		start = t
	case xml.CharData:
		if len(bytes.TrimLeft(t, " \t\r\n")) != 0 {
			// Whitespace is allowed, but anything else at the top of the stream is
			// disallowed.
			return stream.BadFormat
		}
		return nil
	default:
		// If this isn't a start element or a whitespace keepalive, the stream is in
		// a bad state.
		return stream.BadFormat
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
		_, id = attr.Get(start.Attr, "id")

		// If this is a response IQ (ie. an "error" or "result") check if we're
		// handling it as part of a SendIQ call.
		// If not, record this so that we can check if the user sends a response
		// later.
		if !iqNeedsResp(start.Attr) {
			s.sentIQMutex.Lock()
			c := s.sentIQs[id]
			s.sentIQMutex.Unlock()
			if c == nil {
				goto noreply
			}

			inner := xmlstream.Inner(r)
			c <- iqResponder{
				r: xmlstream.MultiReader(xmlstream.Token(start), inner, xmlstream.Token(start.End())),
				c: c,
			}
			<-c
			// Consume the rest of the stream before continuing the loop.
			_, err = xmlstream.Copy(discard, inner)
			if err != nil {
				return err
			}
			return nil
		}
		needsResp = true
	}

noreply:

	w := s.TokenWriter()
	defer w.Close()
	rw := &responseChecker{
		TokenReader: xmlstream.MultiReader(xmlstream.Inner(r), xmlstream.Token(start.End())),
		TokenWriter: w,
		id:          id,
	}
	if err := handler.HandleXMPP(rw, &start); err != nil {
		return err
	}

	// If the user did not write a response to an IQ, send a default one.
	if needsResp && !rw.wroteResp {
		_, err := xmlstream.Copy(w, stanza.IQ{
			ID:   id,
			Type: stanza.ErrorIQ,
		}.Wrap(stanza.Error{
			Type:      stanza.Cancel,
			Condition: stanza.ServiceUnavailable,
		}.TokenReader()))
		if err != nil {
			return err
		}
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// Advance to the end of the current element before attempting to read the
	// next.
	_, err = xmlstream.Copy(discard, rw)
	return err
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
		_, id := attr.Get(tok.Attr, "id")
		if rw.level < 1 && isIQEmptySpace(tok.Name) && id == rw.id && !iqNeedsResp(tok.Attr) {
			rw.wroteResp = true
		}
		rw.level++
	case xml.EndElement:
		rw.level--
	}

	return rw.TokenWriter.EncodeToken(t)
}

func (rw *responseChecker) Encode(v interface{}) error {
	return marshal.EncodeXML(rw, v)
}

func (rw *responseChecker) EncodeElement(v interface{}, start xml.StartElement) error {
	return marshal.EncodeXMLElement(rw, v, start)
}

// Feature checks if a feature with the given namespace was advertised
// by the server for the current stream. If it was data will be the canonical
// representation of the feature as returned by the feature's Parse function.
func (s *Session) Feature(namespace string) (data interface{}, ok bool) {
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

type lockWriteCloser struct {
	w   *Session
	err error
	m   sync.Locker
}

func (lwc *lockWriteCloser) EncodeToken(t xml.Token) error {
	if lwc.err != nil {
		return lwc.err
	}

	lwc.w.stateMutex.RLock()
	if lwc.w.state&OutputStreamClosed == OutputStreamClosed {
		lwc.w.stateMutex.RUnlock()
		return ErrOutputStreamClosed
	}
	lwc.w.stateMutex.RUnlock()

	return lwc.w.out.e.EncodeToken(t)
}

func (lwc *lockWriteCloser) Flush() error {
	if lwc.err != nil {
		return nil
	}
	lwc.w.stateMutex.RLock()
	if lwc.w.state&OutputStreamClosed == OutputStreamClosed {
		lwc.w.stateMutex.RUnlock()
		return ErrOutputStreamClosed
	}
	lwc.w.stateMutex.RUnlock()
	return lwc.w.out.e.Flush()
}

func (lwc *lockWriteCloser) Close() error {
	if lwc.err != nil {
		return nil
	}
	lwc.err = io.EOF
	lwc.m.Unlock()
	return nil
}

type lockReadCloser struct {
	s   *Session
	err error
	m   sync.Locker
}

func (lrc *lockReadCloser) Token() (xml.Token, error) {
	if lrc.err != nil {
		return nil, lrc.err
	}

	lrc.s.stateMutex.RLock()
	if lrc.s.state&InputStreamClosed == InputStreamClosed {
		lrc.s.stateMutex.RUnlock()
		return nil, ErrInputStreamClosed
	}
	lrc.s.stateMutex.RUnlock()

	return lrc.s.in.d.Token()
}

func (lrc *lockReadCloser) Close() error {
	if lrc.err != nil {
		return nil
	}
	lrc.err = io.EOF
	lrc.m.Unlock()
	return nil
}

// TokenWriter returns a new xmlstream.TokenWriteCloser that can be used to
// write raw XML tokens to the session.
// All other writes and future calls to TokenWriter will block until the Close
// method is called.
// After the TokenWriteCloser has been closed, any future writes will return
// io.EOF.
func (s *Session) TokenWriter() xmlstream.TokenWriteFlushCloser {
	s.out.Lock()

	return &lockWriteCloser{
		m: s.out.Locker,
		w: s,
	}
}

// TokenReader returns a new xmlstream.TokenReadCloser that can be used to read
// raw XML tokens from the session.
// All other reads and future calls to TokenReader will block until the Close
// method is called.
// After the TokenReadCloser has been closed, any future reads will return
// io.EOF.
func (s *Session) TokenReader() xmlstream.TokenReadCloser {
	s.in.Lock()

	return &lockReadCloser{
		m: s.in.Locker,
		s: s,
	}
}

// Close ends the output stream (by sending a closing </stream:stream> token).
// It does not close the underlying connection.
// Calling Close() multiple times will only result in one closing
// </stream:stream> being sent.
func (s *Session) Close() error {
	s.out.Lock()
	defer s.out.Unlock()
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	return s.closeSession()
}

func (s *Session) closeSession() error {
	if s.state&OutputStreamClosed == OutputStreamClosed {
		return nil
	}

	s.state |= OutputStreamClosed
	// We wrote the opening stream instead of encoding it, so do the same with the
	// closing to ensure that the encoder doesn't think the tokens are mismatched.
	_, err := s.Conn().Write([]byte(closeStreamTag))
	return err
}

// State returns the current state of the session. For more information, see the
// SessionState type.
func (s *Session) State() SessionState {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	return s.state
}

// LocalAddr returns the Origin address for initiated connections, or the
// Location for received connections.
func (s *Session) LocalAddr() jid.JID {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	if (s.state & Received) == Received {
		return s.location
	}
	return s.origin
}

// RemoteAddr returns the Location address for initiated connections, or the
// Origin address for received connections.
func (s *Session) RemoteAddr() jid.JID {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
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

// EncodeIQ is like Encode except that it returns an error if v does not marshal
// to an IQ stanza and like SendIQ it blocks until a response is received.
func (s *Session) EncodeIQ(ctx context.Context, v interface{}) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(v)
	if err != nil {
		return nil, err
	}
	return s.SendIQ(ctx, r)
}

// EncodeIQElement is like EncodeIQ except that it wraps the payload in an
// Info/Query (IQ) element.
func (s *Session) EncodeIQElement(ctx context.Context, payload interface{}, iq stanza.IQ) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(payload)
	if err != nil {
		return nil, err
	}
	return s.SendIQElement(ctx, r, iq)
}

// Encode writes the XML encoding of v to the stream.
//
// For more information see "encoding/xml".Encode.
func (s *Session) Encode(v interface{}) error {
	s.out.Lock()
	defer s.out.Unlock()

	return marshal.EncodeXML(s.out.e, v)
}

// EncodeElement writes the XML encoding of v to the stream, using start as the
// outermost tag in the encoding.
//
// For more information see "encoding/xml".EncodeElement.
func (s *Session) EncodeElement(v interface{}, start xml.StartElement) error {
	s.out.Lock()
	defer s.out.Unlock()

	return marshal.EncodeXMLElement(s.out.e, v, start)
}

// Send transmits the first element read from the provided token reader.
//
// Send is safe for concurrent use by multiple goroutines.
func (s *Session) Send(ctx context.Context, r xml.TokenReader) error {
	return send(s, r, nil)
}

// SendElement is like Send except that it uses start as the outermost tag in
// the encoding and uses the entire token stream as the payload.
//
// SendElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendElement(ctx context.Context, r xml.TokenReader, start xml.StartElement) error {
	return send(s, r, &start)
}

func send(s *Session, r xml.TokenReader, start *xml.StartElement) error {
	s.out.Lock()
	defer s.out.Unlock()

	if start == nil {
		tok, err := r.Token()
		if err != nil {
			return err
		}

		el, ok := tok.(xml.StartElement)
		if !ok {
			return errNotStart
		}
		start = &el
		r = xmlstream.Inner(r)
	}

	err := s.out.e.EncodeToken(*start)
	if err != nil {
		return err
	}
	_, err = xmlstream.Copy(s.out.e, r)
	if err != nil {
		return err
	}
	err = s.out.e.EncodeToken(start.End())
	if err != nil {
		return err
	}
	return s.out.e.Flush()
}

func iqNeedsResp(attrs []xml.Attr) bool {
	var typ string
	for _, attr := range attrs {
		if attr.Name.Local == "type" {
			typ = attr.Value
			break
		}
	}

	return typ == string(stanza.GetIQ) || typ == string(stanza.SetIQ)
}

func isIQ(name xml.Name) bool {
	return name.Local == "iq" && (name.Space == ns.Client || name.Space == ns.Server)
}

func isIQEmptySpace(name xml.Name) bool {
	return name.Local == "iq" && (name.Space == "" || name.Space == ns.Client || name.Space == ns.Server)
}

func isStanza(name xml.Name) bool {
	return (name.Local == "iq" || name.Local == "message" || name.Local == "presence") &&
		(name.Space == ns.Client || name.Space == ns.Server)
}

// SendIQ is like Send except that it returns an error if the first token read
// from the stream is not an Info/Query (IQ) start element and blocks until a
// response is received.
//
// If the input stream is not being processed (a call to Serve is not running),
// SendIQ will never receive a response and will block until the provided
// context is canceled.
// If the response is non-nil, it does not need to be consumed in its entirety,
// but it must be closed before stream processing will resume.
// If the IQ type does not require a response—ie. it is a result or error IQ,
// meaning that it is a response itself—SendIQElemnt does not block and the
// response is nil.
//
// If the context is closed before the response is received, SendIQ immediately
// returns the context error.
// Any response received at a later time will not be associated with the
// original request but can still be handled by the Serve handler.
//
// If an error is returned, the response will be nil; the converse is not
// necessarily true.
// SendIQ is safe for concurrent use by multiple goroutines.
func (s *Session) SendIQ(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error) {
	tok, err := r.Token()
	if err != nil {
		return nil, err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil, fmt.Errorf("expected IQ start element, got %T", tok)
	}
	if !isIQEmptySpace(start.Name) {
		return nil, fmt.Errorf("expected start element to be an IQ")
	}

	// If there's no ID, add one.
	idx, id := attr.Get(start.Attr, "id")
	if idx == -1 {
		idx = len(start.Attr)
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: ""})
	}
	if id == "" {
		id = attr.RandomID()
		start.Attr[idx].Value = id
	}

	// If this an IQ of type "set" or "get" we expect a response.
	if iqNeedsResp(start.Attr) {
		// return s.sendResp(ctx, id, xmlstream.Wrap(r, start))
		return s.sendResp(ctx, id, xmlstream.Inner(r), start)
	}

	// If this is an IQ of type result or error, we don't expect a response so
	// just send it normally.
	return nil, s.SendElement(ctx, xmlstream.Inner(r), start)
}

// SendIQElement is like SendIQ except that it wraps the payload in an
// Info/Query (IQ) element.
// For more information, see SendIQ.
//
// SendIQElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendIQElement(ctx context.Context, payload xml.TokenReader, iq stanza.IQ) (xmlstream.TokenReadCloser, error) {
	return s.SendIQ(ctx, iq.Wrap(payload))
}

func (s *Session) sendResp(ctx context.Context, id string, payload xml.TokenReader, start xml.StartElement) (xmlstream.TokenReadCloser, error) {
	c := make(chan xmlstream.TokenReadCloser)

	s.sentIQMutex.Lock()
	s.sentIQs[id] = c
	s.sentIQMutex.Unlock()
	defer func() {
		s.sentIQMutex.Lock()
		delete(s.sentIQs, id)
		s.sentIQMutex.Unlock()
	}()

	err := s.SendElement(ctx, payload, start)
	if err != nil {
		return nil, err
	}

	select {
	case rr := <-c:
		return rr, nil
	case <-ctx.Done():
		close(c)
		return nil, ctx.Err()
	}
}

// closeInputStream immediately marks the input stream as closed and cancels any
// deadlines associated with it.
func (s *Session) closeInputStream() {
	s.in.Lock()
	defer s.in.Unlock()
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
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
						Value: attr.RandomID(),
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
