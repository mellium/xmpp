// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package receipts implements XEP-0184: Message Delivery Receipts.
package receipts // import "mellium.im/xmpp/receipts"

import (
	"context"
	"encoding/xml"
	"fmt"
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

	r := xmlstream.Wrap(nil, xml.StartElement{
		Name: xml.Name{Space: NS, Local: "request"},
	})
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
