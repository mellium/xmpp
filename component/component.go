// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package component is used to establish XEP-0114: Jabber Component Protocol
// connections.
package component // import "mellium.im/xmpp/component"

import (
	"context"
	/* #nosec */
	"crypto/sha1"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

// A list of namespaces used by this package, provided as a convenience.
const (
	NSAccept = `jabber:component:accept`
)

// NewClientSession initiates an XMPP session on the given io.ReadWriter using
// the component protocol.
func NewClientSession(ctx context.Context, addr jid.JID, secret []byte, rw io.ReadWriter, received bool) (*xmpp.Session, error) {
	addr = addr.Domain()
	return xmpp.NegotiateSession(ctx, addr, addr, rw, received, Negotiator(addr, secret, false))
}

// AcceptSession accepts an XMPP session on the given io.ReadWriter using the
// component protocol.
//func AcceptSession(ctx context.Context, addr jid.JID, secret []byte, rw io.ReadWriter) (*xmpp.Session, error) {
//	return xmpp.NegotiateSession(ctx, nil, rw, Negotiator(addr, secret, true))
//}

// Negotiator returns a new function that can be used to negotiate a component
// protocol connection when passed to xmpp.NegotiateSession.
//
// It currently only supports the client side of the component protocol.
// If recv is true (indicating that we are receiving a connection on the server
// side) the returned xmpp.Negotiator will panic.
func Negotiator(addr jid.JID, secret []byte, recv bool) xmpp.Negotiator {
	return func(ctx context.Context, s *xmpp.Session, _ interface{}) (mask xmpp.SessionState, _ io.ReadWriter, _ interface{}, err error) {
		d := xml.NewDecoder(s.Conn())

		if recv {
			// If we're the receiving entity wait for a new stream, then send one in
			// response.
			panic("component: receiving connections not yet implemented")
		} else {
			// If we're the initiating entity, send a new stream and then wait for one
			// in response.
			_, err = fmt.Fprintf(s.Conn(), `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='%s'>`, addr)
			if err != nil {
				return mask, nil, nil, err
			}
		}

		foundProc := false
		var start xml.StartElement
		// TODO: This loop is stupid and probably broken. Find a way to reuse existing
		// logic from the xmpp package?
	procloop:
		for {
			tok, err := d.Token()
			if err != nil {
				return mask, nil, nil, err
			}
			switch t := tok.(type) {
			case xml.ProcInst:
				if !foundProc {
					foundProc = true
					continue
				}
				return mask, nil, nil, errors.New("Received unexpected proc inst from server")
			case xml.StartElement:
				start = t
				break procloop
			default:
				return mask, nil, nil, errors.New("Received unexpected token from server")
			}
		}

		if start.Name.Local != "stream" || start.Name.Space != stream.NS {
			return mask, nil, nil, errors.New("Expected stream:stream from server")
		}

		var id string
		for _, attr := range start.Attr {
			if attr.Name.Local == "id" {
				id = attr.Value
				break
			}
		}

		if id == "" {
			return mask, nil, nil, errors.New("Expected server stream to contain stream ID")
		}

		/* #nosec */
		h := sha1.New()

		// hash.Write never returns an error per the documentation.
		/* #nosec */
		_, _ = h.Write([]byte(id))

		// hash.Write never returns an error per the documentation.
		/* #nosec */
		_, _ = h.Write(secret)

		_, err = fmt.Fprintf(s.Conn(), `<handshake>%x</handshake>`, h.Sum(nil))
		if err != nil {
			return mask, nil, nil, err
		}

		tok, err := d.Token()
		if err != nil {
			return mask, nil, nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			return mask, nil, nil, errors.New("Expected acknowledgement or error start token from server")
		}

		// TODO: Actually unmarshal the stream error.
		switch start.Name.Local {
		case "error":
			return mask, nil, nil, stream.NotAuthorized
		case "handshake":
			err = d.Skip()
			return xmpp.Ready | xmpp.Authn, nil, nil, err
		}

		return mask, nil, nil, fmt.Errorf("Unknown start element: %v", start)
	}
}
