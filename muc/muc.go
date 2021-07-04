// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package muc implements Multi-User Chat.
package muc // import "mellium.im/xmpp/muc"

import (
	"context"
	"encoding/xml"
	"sync"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/internal/attr"
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
		// TODO: make consts for the statuses when possible. Wait until we can
		// determine if they can be generated or have to be hand rolled first.
		// See: https://github.com/xsf/registrar/pull/38
		if decodedPresence.HasStatus(110) && channel.join != nil {
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
	if p.Type != "" {
		p.Type = ""
	}
	if p.ID == "" {
		p.ID = attr.RandomID()
	}

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

	conf := config{}
	for _, o := range opt {
		o(&conf)
	}
	channel.pass = conf.password

	errChan := make(chan error)
	go func(errChan chan<- error) {
		resp, err := s.SendPresenceElement(ctx, conf.TokenReader(), p)
		//err := s.Send(ctx, p.Wrap(conf.TokenReader()))
		if err != nil {
			errChan <- err
			return
		}
		/* #nosec */
		defer resp.Close()
		// Pop the start presence token.
		_, err = resp.Token()
		if err != nil {
			errChan <- err
			return
		}

		stanzaError, err := stanza.UnmarshalError(resp)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- stanzaError
	}(errChan)

	select {
	case err := <-errChan:
		return nil, err
	case roomAddr := <-channel.join:
		channel.addr = roomAddr
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return channel, nil
}
