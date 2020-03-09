// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ping implements XEP-0199: XMPP Ping.
package ping // import "mellium.im/xmpp/ping"

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// NS is the XML namespace used by XMPP pings. It is provided as a convenience.
const NS = `urn:xmpp:ping`

// Handle returns an option that registers a Handler for ping requests.
func Handle() mux.Option {
	return mux.IQ(stanza.GetIQ, xml.Name{Local: "ping", Space: NS}, Handler{})
}

// Handler responds to ping requests.
type Handler struct{}

// HandleIQ implements mux.IQHandler.
func (h Handler) HandleIQ(iq stanza.IQ, t xmlstream.DecodeEncoder, start *xml.StartElement) error {
	if iq.Type != stanza.GetIQ || start.Name.Local != "ping" || start.Name.Space != NS {
		return nil
	}

	_, err := xmlstream.Copy(t, iq.Result(nil))
	return err
}

// Send sends a ping to the provided JID and blocks until a response is
// received.
// Pings sent to other clients should use the full JID, otherwise they will be
// handled by the server.
//
// If the remote JID reports that the ping service is unavailable, no error is
// returned because we were able to receive the error response (the remote
// resource exists and could be pinged, it just doesn't support this particular
// protocol for doing so).
func Send(ctx context.Context, s *xmpp.Session, to jid.JID) error {
	iq := IQ{IQ: stanza.IQ{
		Type: stanza.GetIQ,
		To:   to,
	}}
	resp, err := s.SendIQ(ctx, iq.TokenReader())
	if resp != nil {
		defer resp.Close()
	}

	if stanzaErr, ok := err.(stanza.Error); ok {
		// If the ping namespace isn't supported and we get back
		// service-unavailable, treat this as if the ping succeeded (because the
		// client was obviously able to send us the error that ping isn't
		// supported).
		if stanzaErr.Condition == stanza.ServiceUnavailable {
			return nil
		}
	}
	return err
}

// IQ is encoded as a ping request.
type IQ struct {
	stanza.IQ

	Ping struct{} `xml:"urn:xmpp:ping ping"`
}

// WriteXML satisfies the xmlstream.WriterTo interface. It is like MarshalXML
// except it writes tokens to w.
func (iq IQ) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (iq IQ) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "ping", Space: NS}}
	return iq.Wrap(xmlstream.Wrap(nil, start))
}
