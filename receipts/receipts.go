// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package receipts implements XEP-0184: Message Delivery Receipts.
package receipts // import "mellium.im/xmpp/receipts"

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const (
	// NS is the XML namespace used by message delivery receipts.
	// It is provided as a convenience.
	NS = "urn:xmpp:receipts"
)

// Requested is a type that can be added to messages to request a read receipt.
// When unmarshaled or marshaled its value indicates whether it was or will be
// present in the message.
//
// This type is used to manually include a request in a message struct.
// To send a message and wait for the receipt see the methods on Handler.
type Requested struct {
	XMLName xml.Name `xml:"urn:xmpp:receipts request"`
	Value   bool
}

// TokenReader implements xmlstream.Marshaler.
func (r Requested) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "request"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (r Requested) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, r.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (r Requested) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := r.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// UnmarshalXML implements xml.Unmarshaler.
func (r *Requested) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	r.Value = start.Name.Space == NS && start.Name.Local == "request"
	return d.Skip()
}

// Request is an xmlstream.Transformer that inserts a request for a read receipt
// into any message read through r that is not itself a receipt.
// It is provided to allow easily requesting read receipts asynchronously.
// To send a message and block waiting on a read receipt, see the methods on
// Handler.
func Request(r xml.TokenReader) xml.TokenReader {
	var (
		noWrite bool
		inner   xml.TokenReader
	)
	return xmlstream.ReaderFunc(func() (xml.Token, error) {
	start:
		if inner != nil {
			tok, err := inner.Token()
			if err == io.EOF {
				inner = nil
				err = nil
			}
			return tok, err
		}

		tok, err := r.Token()
		switch err {
		case io.EOF:
			if tok == nil {
				return nil, err
			}
			err = nil
		case nil:
		default:
			return tok, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Local == "receipt" && t.Name.Space == NS:
				noWrite = true
			case t.Name.Local == "message" && (t.
				Name.Space == ns.Client || t.Name.Space == ns.Server):
				noWrite = false
				for _, attr := range t.Attr {
					if attr.Name.Local == "type" {
						noWrite = attr.Value == "error"
						break
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "message" && (t.Name.Space == ns.Client || t.Name.Space == ns.Server) {
				if !noWrite {
					inner = xmlstream.MultiReader(xmlstream.Wrap(nil, xml.StartElement{
						Name: xml.Name{Space: NS, Local: "request"},
					}), xmlstream.Token(t))

					goto start
				}
				noWrite = false
			}
		}

		return tok, err
	})
}

// Handle returns an option that registers a Handler for message receipts.
func Handle(h *Handler) mux.Option {
	return func(m *mux.ServeMux) {
		received := xml.Name{Local: "received", Space: NS}
		request := xml.Name{Local: "request", Space: NS}

		// We respond to incoming requests for any message type except error
		// messages. We do, however, match up error messages with their send calls
		// if the user manually sent one.
		mux.Message(stanza.NormalMessage, received, h)(m)
		mux.Message(stanza.NormalMessage, request, h)(m)
		mux.Message(stanza.ChatMessage, received, h)(m)
		mux.Message(stanza.ChatMessage, request, h)(m)
		mux.Message(stanza.HeadlineMessage, received, h)(m)
		mux.Message(stanza.HeadlineMessage, request, h)(m)
		mux.Message(stanza.GroupChatMessage, received, h)(m)
		mux.Message(stanza.GroupChatMessage, request, h)(m)
		mux.Message(stanza.ErrorMessage, received, h)(m)
	}
}

// Handler listens for incoming message receipts and matches them to outgoing
// messages sent with SendMessage or SendMessageElement.
type Handler struct {
	sent map[string]chan struct{}
	m    sync.Mutex
}

// HandleMessage implements mux.MessageHandler and responds to requests and
// responses for message delivery receipts.
func (h *Handler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	// Pop the start message token
	_, err := t.Token()
	if err != nil {
		return err
	}

	i := xmlstream.NewIter(t)
	/* #nosec */
	defer i.Close()

	for i.Next() {
		start, _ := i.Current()
		switch start.Name.Local {
		case "received":
			_, id := attr.Get(start.Attr, "id")
			h.m.Lock()
			c, ok := h.sent[id]
			if ok {
				delete(h.sent, id)
				h.m.Unlock()
			} else {
				h.m.Unlock()
				return nil
			}

			c <- struct{}{}
			return nil
		case "request":
			msg.From, msg.To = msg.To, msg.From
			id := msg.ID
			msg.ID = ""

			_, err = xmlstream.Copy(t, msg.Wrap(xmlstream.Wrap(nil, xml.StartElement{
				Name: xml.Name{Space: NS, Local: "received"},
				Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: id}},
			})))
			return err
		}
	}
	return i.Err()
}

// SendMessage transmits the first element read from the provided token reader
// over the session if the element is a message stanza, otherwise it returns an
// error.
// SendMessage adds a request for a message receipt and an ID if one does not
// already exist.
//
// If the context is closed before the message delivery receipt is received,
// SendMessage immediately returns the context error.
// Any response received at a later time will no be associated with the original
// request, but can still be handled by the Handler.
// If the returned error is nil, receipt of the message was successfully
// acknowledged.
//
// SendMessage is safe for concurrent use by multiple goroutines.
func (h *Handler) SendMessage(ctx context.Context, s *xmpp.Session, r xml.TokenReader) error {
	tok, err := r.Token()
	if err != nil {
		return err
	}
	start := tok.(xml.StartElement)
	if start.Name.Local != "message" || (start.Name.Space != ns.Server && start.Name.Space != ns.Client) {
		return fmt.Errorf("expected a message type, got %v", start.Name)
	}
	msg, err := stanza.NewMessage(start)
	if err != nil {
		return err
	}

	return h.SendMessageElement(ctx, s, xmlstream.Inner(r), msg)
}

// SendMessageElement is like SendMessage except that it wraps the payload in
// the message element derived from msg.
// For more information, see SendMessage.
//
// SendMessageElement is safe for concurrent use by multiple goroutines.
func (h *Handler) SendMessageElement(ctx context.Context, s *xmpp.Session, payload xml.TokenReader, msg stanza.Message) error {
	if h.sent == nil {
		h.m.Lock()
		h.sent = make(map[string]chan struct{})
		h.m.Unlock()
	}

	if msg.ID == "" {
		msg.ID = attr.RandomID()
	}

	c := make(chan struct{})
	h.m.Lock()
	h.sent[msg.ID] = c
	h.m.Unlock()

	r := Requested{Value: true}.TokenReader()
	if payload != nil {
		r = xmlstream.MultiReader(payload, r)
	}
	err := s.SendElement(ctx, r, msg.StartElement())
	if err != nil {
		return err
	}

	select {
	case <-c:
		return nil
	case <-ctx.Done():
		h.m.Lock()
		delete(h.sent, msg.ID)
		h.m.Unlock()
		close(c)
		return ctx.Err()
	}
}
