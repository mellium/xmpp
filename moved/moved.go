// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package moved provides a mechanism for moving from one server to another.
//
// BE ADVISED: this package is experimental and subject to breaking changes or
// being removed entirely without warning.
package moved // import "mellium.im/xmpp/moved"

import (
	"context"
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

// NS is the namespace supported by this package, provided as a convenience.
const NS = "urn:xmpp:moved:1"

// Move informs a users contacts that they have moved their account from the
// account logged in at oldSession to the account logged in at newSession.
func Move(ctx context.Context, oldSession *xmpp.Session, newSession *xmpp.Session) error {
	// TODO: implement PEP and use that.
	payload := xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Wrap(
				xmlstream.Wrap(
					xmlstream.Wrap(
						xmlstream.Token(newSession.LocalAddr().Bare().String()),
						xml.StartElement{Name: xml.Name{Local: "new-jid"}},
					),
					xml.StartElement{Name: xml.Name{Space: NS, Local: "moved"}},
				),
				xml.StartElement{Name: xml.Name{Local: "item"}, Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "current"}}},
			),
			xml.StartElement{Name: xml.Name{Space: NS, Local: "publish"}},
		),
		xml.StartElement{Name: xml.Name{Space: "http://jabber.org/protocol/pubsub", Local: "pubsub"}},
	)
	r, err := oldSession.SendIQElement(ctx, payload, stanza.IQ{})
	if err != nil {
		return err
	}
	// TODO: does SendIQElement unmarshal errors?
	err = r.Close()
	if err != nil {
		return err
	}

	// TODO: provide a second function to do this with an existing cached roster so
	// we don't have to query for it.
	iter := roster.Fetch(ctx, oldSession)
	/* #nosec */
	defer iter.Close()
	for iter.Next() {
		item := iter.Item()
		// TODO: how have we gotten this far without some simple subscription
		// request functionality? Implement a presence package or similar with this
		// sort of thing.
		newSession.Send(ctx, stanza.Presence{
			Type: stanza.SubscribePresence,
			To:   item.JID,
		}.Wrap(xmlstream.Wrap(
			xmlstream.Wrap(
				xmlstream.Token(oldSession.LocalAddr().Bare().String()),
				xml.StartElement{Name: xml.Name{Local: "old-jid"}},
			),
			xml.StartElement{Name: xml.Name{Space: NS, Local: "moved"}},
		)))
	}
	return iter.Err()
}

// Handle returns a mux option for incoming moved requests.
func Handle(h Handler) mux.Option {
	return mux.Presence(stanza.SubscribePresence, xml.Name{Space: NS, Local: "moved"}, h)
}

// NewHandler creates a handler for verifying moved requests.
// It calls f with the results including the new JID and a boolean indicating
// whether it could be confirmed against the old JID.
//
// Any errors returned from f will be passed through and returned from the call
// to HandlePresence.
func NewHandler(s *xmpp.Session, f func(from jid.JID, ok bool) error) Handler {
	return Handler{
		s: s,
		f: f,
	}
}

// Handler receives and verifies incoming moved requests.
type Handler struct {
	S       *xmpp.Session
	Timeout time.Duration

	// F is called with the results including the new JID and a boolean indicating
	// whether it could be confirmed by the old JID.
	F func(jid.JID, bool) error
}

// HandlePresence implements mux.PresenceHandler.
func (h Handler) HandlePresence(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
	// Pop presence token
	_, err := r.Token()
	if err != nil {
		return err
	}
	d := xml.NewTokenDecoder(r)
	s := struct {
		XMLName xml.Name `xml:"urn:xmpp:moved:1 moved"`
		oldJID  jid.JID  `xml:"old-jid"`
	}{}
	err = d.Decode(&s)
	if err != nil {
		return err
	}

	payload := xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Local: "item"}, Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "current"}}},
			),
			xml.StartElement{Name: xml.Name{Space: NS, Local: "items"}},
		),
		xml.StartElement{Name: xml.Name{Space: "http://jabber.org/protocol/pubsub", Local: "pubsub"}},
	)
	ctx := context.Background()
	if h.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.Timeout)
		defer cancel()
	}
	resp, err := h.s.SendIQElement(ctx, payload, stanza.IQ{
		To: s.oldJID.Bare(),
	})
	if err != nil {
		return err
	}
	/* #nosec */
	defer resp.Close()
	d = xml.NewTokenDecoder(resp)
	newJIDResp := struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
		newJID  jid.JID  `xml:">items>item>moved>new-jid"`
	}{}
	err = d.Decode(&newJIDResp)
	if err != nil {
		return err
	}

	if h.f != nil {
		return h.f(newJIDResp.newJID, p.From.Equal(newJIDResp.newJID))
	}
	return nil
}
