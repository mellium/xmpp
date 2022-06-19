// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -receiver "*Client"

// Package muc implements Multi-User Chat.
//
// Unlike many Multi-User Chat (MUC) implementations, the muc package tries to
// be as stateless as possible.
// It allows you to receive chat messages and invites sent through a channel,
// for example, but does not keep track of what users are joined to the channel
// at any given time.
// This is best left up to the user who may want to use a distributed datastore
// to keep track of users in a large system for searching many public channels,
// or may want a simple in-memory map for a small client.
//
// The main entrypoint into the muc package (for clients) is the Client type.
// It can be used to join MUCs and has callbacks for receiving MUC events such
// as presence or mediated invites to a new channel.
// It is normally registered with a multiplexer such as the one found in the mux
// package:
//
//	mucClient := muc.Client{}
//	m := mux.New(
//	    muc.HandleClient(mucClient),
//	)
//	channel, err := mucClient.Join(…)
//
// Once the Join method has been called the resulting channel type can be used
// to perform actions on the channel such as setting the subject, rejoining (to
// force syncronize state), or leaving the channel.
//
//	channel, err := mucClient.Join(…)
//	channel.Subject(context.Background(), "Bridge operation and tactical readiness")
package muc // import "mellium.im/xmpp/muc"

import (
	"context"
	"encoding/xml"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Various namespaces used by this package, provided as a convenience.
const (
	NS      = `http://jabber.org/protocol/muc`
	NSUser  = `http://jabber.org/protocol/muc#user`
	NSOwner = `http://jabber.org/protocol/muc#owner`
	NSAdmin = `http://jabber.org/protocol/muc#admin`

	// NSConf is the legacy conference namespace, now only used for direct MUC
	// invitations and backwards compatibility.
	NSConf = `jabber:x:conference`
)

// GetConfig requests a room config form.
func GetConfig(ctx context.Context, room jid.JID, s *xmpp.Session) (*form.Data, error) {
	return GetConfigIQ(ctx, stanza.IQ{
		To: room,
	}, s)
}

// GetConfigIQ is like GetConfig except that it lets you customize the IQ.
// Changing the type of the IQ has no effect.
func GetConfigIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) (*form.Data, error) {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	formResp := struct {
		XMLName  xml.Name  `xml:"http://jabber.org/protocol/muc#owner query"`
		DataForm form.Data `xml:"jabber:x:data x"`
	}{
		DataForm: form.Data{},
	}
	err := s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NSOwner, Local: "query"}},
	), iq, &formResp)
	return &formResp.DataForm, err
}

// SetConfig sets the room config.
// The form should be the one provided by a call to GetConfig with various
// values set.
func SetConfig(ctx context.Context, room jid.JID, form *form.Data, s *xmpp.Session) error {
	return SetConfigIQ(ctx, stanza.IQ{
		To: room,
	}, form, s)
}

// SetConfigIQ is like SetConfig except that it lets you customize the IQ.
// Changing the type of the IQ has no effect.
func SetConfigIQ(ctx context.Context, iq stanza.IQ, form *form.Data, s *xmpp.Session) error {
	if iq.Type != stanza.SetIQ {
		iq.Type = stanza.SetIQ
	}
	submission, _ := form.Submit()
	r, err := s.SendIQElement(ctx, xmlstream.Wrap(
		submission,
		xml.StartElement{Name: xml.Name{Space: NSOwner, Local: "query"}},
	), iq)
	if err != nil {
		return err
	}
	return r.Close()
}

// HandleClient returns an option that registers the handler for use with a
// multiplexer.
func HandleClient(h *Client) mux.Option {
	return func(m *mux.ServeMux) {
		userPresence := xml.Name{Space: NSUser, Local: "x"}

		mux.Presence(stanza.AvailablePresence, userPresence, h)(m)
		mux.Presence(stanza.UnavailablePresence, userPresence, h)(m)
		mux.Message(stanza.NormalMessage, userPresence, h)(m)
	}
}

// Client is an xmpp.Handler that handles MUC payloads from a client
// perspective.
type Client struct {
	managed  map[string]*Channel
	managedM sync.Mutex

	// HandleInvite will be called if we receive a mediated MUC invitation.
	HandleInvite       func(Invitation)
	HandleUserPresence func(stanza.Presence, Item)
}

// HandleMessage satisfies mux.MessageHandler.
// it is used by the multiplexer and normally does not need to be called by the
// user.
func (c *Client) HandleMessage(p stanza.Message, r xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(r)
	msg := struct {
		stanza.Message
		X Invitation `xml:"http://jabber.org/protocol/muc#user x"`
	}{}
	err := d.Decode(&msg)
	if err != nil {
		return err
	}

	if msg.X.XMLName.Local != "" && c.HandleInvite != nil {
		c.HandleInvite(msg.X)
		return nil
	}
	return nil
}

type mucPresence struct {
	stanza.Presence
	X struct {
		XMLName xml.Name
		Item    Item `xml:"item"`
		Status  []struct {
			Code int `xml:"code,attr"`
		} `xml:"status,omitempty"`
	} `xml:"x"`
}

func (p *mucPresence) HasStatus(code int) bool {
	for _, status := range p.X.Status {
		if status.Code == code {
			return true
		}
	}
	return false
}

// HandlePresence satisfies mux.PresenceHandler.
// it is used by the multiplexer and normally does not need to be called by the
// user.
func (c *Client) HandlePresence(p stanza.Presence, r xmlstream.TokenReadEncoder) error {
	// If this is a self-presence, check if we're joining or departing and send on
	// the channel.
	c.managedM.Lock()
	defer c.managedM.Unlock()
	channel, ok := c.managed[p.From.String()]
	// TODO: what do we do with presences that aren't managed?
	if !ok {
		return nil
	}
	d := xml.NewTokenDecoder(r)
	var decodedPresence mucPresence
	err := d.Decode(&decodedPresence)
	if err != nil {
		return err
	}

	switch p.Type {
	case stanza.AvailablePresence:
		if channel.join != nil {
			channel.join <- p.From
			channel.join = nil
			return nil
		}
		if decodedPresence.X.XMLName.Space == NSUser && c.HandleUserPresence != nil {
			c.HandleUserPresence(decodedPresence.Presence, decodedPresence.X.Item)
		}
	case stanza.UnavailablePresence:
		channel.depart <- struct{}{}
		delete(c.managed, channel.addr.String())
	}
	return nil
}

// Join a MUC on the provided session.
// Room should be a full JID in which the desired nickname is the resourcepart.
//
// Join blocks until the full room roster has been received.
func (c *Client) Join(ctx context.Context, room jid.JID, s *xmpp.Session, opt ...Option) (*Channel, error) {
	return c.JoinPresence(ctx, stanza.Presence{
		To: room,
	}, s, opt...)
}

// JoinPresence is like Join except that it gives you more control over the
// presence.
// Changing the presence type has no effect.
func (c *Client) JoinPresence(ctx context.Context, p stanza.Presence, s *xmpp.Session, opt ...Option) (*Channel, error) {
	c.managedM.Lock()

	channel := &Channel{
		addr:    p.To,
		client:  c,
		session: s,

		join:   make(chan jid.JID, 1),
		depart: make(chan struct{}),
	}
	if c.managed == nil {
		c.managed = make(map[string]*Channel)
	}
	c.managed[p.To.String()] = channel
	c.managedM.Unlock()

	err := channel.JoinPresence(ctx, p, opt...)
	return channel, err
}
