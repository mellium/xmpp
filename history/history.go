// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history

import (
	"context"
	"encoding/xml"
	"fmt"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Handle returns an option that registers a Handler for incoming history query
// results.
func Handle(h *Handler) mux.Option {
	return func(m *mux.ServeMux) {
		mux.Message("", xml.Name{Space: NS, Local: "result"}, h)(m)
	}
}

// NewHandler returns a handler capable of handling messages sent from an
// archive.
// Any messages that are part of untracked queries will be passed to the inner
// handler.
func NewHandler(inner mux.MessageHandler) *Handler {
	return &Handler{
		inner:   inner,
		tracked: make(map[string]*Iter),
	}
}

// Handler handles incoming messages from an archive and either passes them on
// to the underlying handler if they are not being tracked or ensures they are
// passed to the correct iterator for syncronous processing if they are.
type Handler struct {
	inner    mux.MessageHandler
	tracked  map[string]*Iter
	trackedM sync.Mutex
}

func (h *Handler) remove(id string) {
	h.trackedM.Lock()
	defer h.trackedM.Unlock()
	if iter, ok := h.tracked[id]; ok {
		close(iter.msgC)
		delete(h.tracked, id)
	}
}

// HandleMessage implements mux.MessageHandler.
func (h *Handler) HandleMessage(msg stanza.Message, r xmlstream.TokenReadEncoder) error {
	// TODO: iterate through and find result.
	msgTok, err := r.Token()
	if err != nil {
		return err
	}
	tok, err := r.Token()
	if err != nil {
		return err
	}
	start := tok.(xml.StartElement)
	var queryID string
	for _, attr := range start.Attr {
		if attr.Name.Local == "queryid" {
			queryID = attr.Value
			break
		}
	}
	h.trackedM.Lock()
	defer h.trackedM.Unlock()
	iter, ok := h.tracked[queryID]
	if !ok {
		if h.inner != nil {
			return h.inner.HandleMessage(msg, struct {
				xml.TokenReader
				xmlstream.Encoder
			}{
				TokenReader: xmlstream.MultiReader(xmlstream.Token(tok), xmlstream.InnerElement(r)),
				Encoder:     r,
			})
		}
		return nil
	}

	iter.msgC <- xmlstream.MultiReader(xmlstream.Token(msgTok), xmlstream.Token(tok), r)
	return nil
}

// Fetch requests messages from the archive and returns an iterator over the
// results.
// Any errors encountered are deferred and returned by the iterator.
func (h *Handler) Fetch(ctx context.Context, filter Query, to jid.JID, s *xmpp.Session) *Iter {
	return h.FetchIQ(ctx, filter, stanza.IQ{
		To: to,
	}, s)
}

// FetchIQ is like Fetch but it allows modifying the underlying IQ.
// Changing the type of the IQ has no effect.
func (h *Handler) FetchIQ(ctx context.Context, filter Query, iq stanza.IQ, s *xmpp.Session) *Iter {
	h.trackedM.Lock()
	defer h.trackedM.Unlock()
	if filter.ID == "" {
		filter.ID = attr.RandomID()
	}
	if _, ok := h.tracked[filter.ID]; ok {
		return &Iter{
			err: fmt.Errorf("history query %s is already being tracked", filter.ID),
		}
	}
	iq.Type = stanza.SetIQ
	msgC := make(chan xml.TokenReader)
	iter := &Iter{
		msgC: msgC,
		h:    h,
		id:   filter.ID,
	}

	go func() {
		var result Result
		err := s.UnmarshalIQ(
			ctx,
			iq.Wrap(filter.TokenReader()),
			&result,
		)
		if err != nil && iter.err != nil {
			// Technically this is racey. I'm not sure that we care though as long as
			// an error is set?
			iter.err = err
		}
		iter.res = result
		h.remove(filter.ID)
	}()

	h.tracked[filter.ID] = iter
	return iter
}

// Fetch requests messages from the archive.
// Messages are received asyncronously and Fetch blocks until the session
// handler has processed them all.
func Fetch(ctx context.Context, filter Query, to jid.JID, s *xmpp.Session) (Result, error) {
	return FetchIQ(ctx, filter, stanza.IQ{
		To: to,
	}, s)
}

// FetchIQ is like Fetch but it allows modifying the underlying IQ.
// Changing the type of the IQ has no effect.
func FetchIQ(ctx context.Context, filter Query, iq stanza.IQ, s *xmpp.Session) (Result, error) {
	if filter.ID == "" {
		filter.ID = attr.RandomID()
	}
	iq.Type = stanza.SetIQ
	var result Result
	err := s.UnmarshalIQ(
		ctx,
		iq.Wrap(filter.TokenReader()),
		&result,
	)
	return result, err
}
