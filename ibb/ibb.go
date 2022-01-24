// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ibb implements data transfer with XEP-0047: In-Band Bytestreams.
//
// In-band bytestreams (IBB) are a bidirectional data transfer mechanism that
// can be used to send small files or transfer other low-bandwidth data.
// Because IBB uses base64 encoding to send the binary data, it is extremely
// inefficient and should only be used as a fallback or last resort.
package ibb // import "mellium.im/xmpp/ibb"

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// NS is the XML namespace used by IBB, provided as a convenience.
const NS = `http://jabber.org/protocol/ibb`

// BlockSize is the default block size used if an IBB stream is opened with no
// block size set.
// Because IBB base64 encodes the underlying data, the actual data transfered
// per stanza will be roughly twice the blocksize.
const BlockSize = 1 << 11

// Handle is an option that registers a handler for all the correct stanza
// types and payloads.
func Handle(h *Handler) mux.Option {
	data := xml.Name{Space: NS, Local: "data"}
	return func(m *mux.ServeMux) {
		mux.Message("", data, h)(m)
		mux.IQ(stanza.SetIQ, data, h)(m)
		mux.IQ(stanza.SetIQ, xml.Name{Space: NS, Local: "open"}, h)(m)
		mux.IQ(stanza.SetIQ, xml.Name{Space: NS, Local: "close"}, h)(m)
	}
}

// Handler is an xmpp.Handler that handles multiplexing of bidirectional IBB
// streams.
type Handler struct {
	mu      sync.Mutex
	streams map[string]*Conn
	l       map[string]Listener
	lM      sync.Mutex
}

// Listen creates a listener that accepts incoming IBB requests.
//
// If a listener has already been created for the given session it is returned
// unaltered.
func (h *Handler) Listen(s *xmpp.Session) Listener {
	addrStr := s.LocalAddr().String()
	if h.l == nil {
		h.l = make(map[string]Listener)
	}
	h.lM.Lock()
	defer h.lM.Unlock()
	l, ok := h.l[addrStr]
	if ok {
		return l
	}
	l = Listener{
		s: s,
		h: h,
		c: make(chan *Conn),
	}
	h.l[addrStr] = l
	return l
}

// HandleMessage implements mux.MessageHandler.
func (h *Handler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(t)
	p := dataMessage{}
	err := d.Decode(&p)
	if err != nil {
		return err
	}
	return handlePayload(h, msg, p.Data, nil)
}

// HandleIQ implements mux.IQHandler.
func (h *Handler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	switch start.Name.Local {
	case "open":
		d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), t))
		p := openPayload{}
		err := d.Decode(&p)
		if err != nil {
			return err
		}
		return handleOpen(h, openIQ{
			IQ:   iq,
			Open: p,
		}, t)
	case "close":
		_, sid := attr.Get(start.Attr, "sid")

		conn, ok := h.streams[sid]
		if !ok {
			_, err := xmlstream.Copy(t, iq.Error(stanza.Error{
				Type:      stanza.Cancel,
				Condition: stanza.ItemNotFound,
			}))
			return err
		}
		err := conn.closeNoNotify(t)
		if err != nil {
			return err
		}
		_, err = xmlstream.Copy(t, iq.Result(nil))
		return err
	case "data":
		d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), t))
		p := dataPayload{}
		err := d.Decode(&p)
		if err != nil {
			return err
		}
		return handlePayload(h, iq, p, t)
	}

	// We understand that this is an IBB payload, but did not recognize the
	// element name so send an error.
	_, err := xmlstream.Copy(t, iq.Error(stanza.Error{
		Type:      stanza.Modify,
		Condition: stanza.FeatureNotImplemented,
	}))
	return err
}

func handleOpen(h *Handler, iq openIQ, e xmlstream.Encoder) error {
	to := iq.To.String()
	h.lM.Lock()
	l, ok := h.l[to]
	if !ok {
		l, ok = h.l[""]
	}
	h.lM.Unlock()
	if !ok {
		_, err := xmlstream.Copy(e, iq.Error(stanza.Error{
			Type:      stanza.Cancel,
			Condition: stanza.NotAcceptable,
		}))
		return err
	}
	_, err := xmlstream.Copy(e, iq.Result(nil))
	if err != nil {
		return err
	}
	conn := newConn(h, l.s, iq, true)
	h.addStream(iq.Open.SID, conn)
	l.c <- conn
	return nil
}

type errorResponder interface {
	Error(stanza.Error) xml.TokenReader
}

func handlePayload(h *Handler, errResp errorResponder, p dataPayload, e xmlstream.Encoder) error {
	conn, ok := h.streams[p.SID]
	if !ok {
		_, err := xmlstream.Copy(e, errResp.Error(stanza.Error{
			Type:      stanza.Cancel,
			Condition: stanza.ItemNotFound,
		}))
		return err
	}

	if p.Seq != conn.seq {
		_, err := xmlstream.Copy(e, errResp.Error(stanza.Error{
			Type:      stanza.Cancel,
			Condition: stanza.UnexpectedRequest,
		}))
		return err
	}
	conn.seq++

	iq, ok := errResp.(stanza.IQ)
	if e != nil && ok {
		_, err := xmlstream.Copy(e, iq.Result(nil))
		if err != nil {
			return err
		}
	}

	conn.readLock.Lock()
	defer conn.readLock.Unlock()
	var inputErr base64.CorruptInputError
	b64Reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(p.Data))
	_, err := conn.readBuf.ReadFrom(b64Reader)
	if errors.As(err, &inputErr) {
		_, err := xmlstream.Copy(e, errResp.Error(stanza.Error{
			Type:      stanza.Cancel,
			Condition: stanza.BadRequest,
		}))
		return err
	}
	if err != nil {
		return err
	}
	// If a call to conn.Read was pending, signal it that it's okay to resume
	// because there's data now.
	select {
	case conn.readReady <- struct{}{}:
	default:
	}
	return nil
}

// Open attempts to create a new IBB stream on the provided session.
func (h *Handler) Open(ctx context.Context, s *xmpp.Session, to jid.JID) (*Conn, error) {
	sid := attr.RandomID()
	return open(ctx, h, true, s, stanza.IQ{To: to}, 0, sid)
}

// OpenIQ is like Open except that it allows you to customize the IQ and other
// properties of the session initialization.
// Changing the type of the IQ has no effect.
func (h *Handler) OpenIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, ack bool, blockSize uint16, sid string) (*Conn, error) {
	return open(ctx, h, ack, s, iq, blockSize, sid)
}

func open(ctx context.Context, h *Handler, acked bool, s *xmpp.Session, start stanza.IQ, blockSize uint16, sid string) (*Conn, error) {
	iq := openIQ{
		IQ: start,
	}
	iq.Type = stanza.SetIQ
	iq.Open.SID = sid
	if acked {
		iq.Open.Stanza = "iq"
	} else {
		iq.Open.Stanza = "message"
	}
	if blockSize == 0 {
		iq.Open.BlockSize = BlockSize
	} else {
		iq.Open.BlockSize = blockSize
	}

	resp, err := s.SendIQ(ctx, iq.TokenReader())
	if err != nil {
		return nil, err
	}
	/* #nosec */
	defer resp.Close()

	conn, err := newConn(h, s, iq, false), nil
	if err != nil {
		return nil, err
	}
	h.addStream(sid, conn)
	return conn, nil
}

func (h *Handler) addStream(sid string, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.streams == nil {
		h.streams = make(map[string]*Conn)
	}
	h.streams[sid] = conn
}

func (h *Handler) rmStream(sid string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.streams, sid)
}
