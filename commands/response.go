// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/stanza"
)

// Response is the response to a command.
// It may contain other commands that can be execute in sequence.
type Response struct {
	stanza.IQ

	Node   string `xml:"node,attr"`
	SID    string `xml:"sessionid,attr"`
	Status string `xml:"status,attr"`
}

// Cancel ends the multi-stage command.
func (r Response) Cancel(ctx context.Context, s *xmpp.Session) error {
	return r.CancelIQ(ctx, stanza.IQ{
		Type: stanza.SetIQ,
		To:   r.IQ.From,
	}, s)
}

// CancelIQ is like Cancel except that it allows you to customize the IQ.
// Changing the type has no effect.
func (r Response) CancelIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) error {
	if iq.Type != stanza.SetIQ {
		iq.Type = stanza.SetIQ
	}
	resp, err := s.SendIQ(ctx, iq.Wrap(Command{
		SID:    r.SID,
		Node:   r.Node,
		Action: "cancel",
	}.TokenReader()))
	if err != nil {
		return err
	}
	/* #nosec */
	defer resp.Close()
	t, err := resp.Token()
	if err != nil {
		return err
	}
	_, err = stanza.UnmarshalIQError(resp, t.(xml.StartElement))
	return err
}

// Complete ends the multi-stage command and optionally submits data.
func (r Response) Complete(ctx context.Context, payload xml.TokenReader, s *xmpp.Session) error {
	return r.CompleteIQ(ctx, stanza.IQ{
		Type: stanza.SetIQ,
		To:   r.IQ.From,
	}, payload, s)
}

// CompleteIQ is like Complete except that it allows you to customize the IQ.
// Changing the type has no effect.
func (r Response) CompleteIQ(ctx context.Context, iq stanza.IQ, payload xml.TokenReader, s *xmpp.Session) error {
	if iq.Type != stanza.SetIQ {
		iq.Type = stanza.SetIQ
	}
	resp, err := s.SendIQ(ctx, iq.Wrap(Command{
		SID:    r.SID,
		Node:   r.Node,
		Action: "complete",
	}.wrap(payload)))
	if err != nil {
		return err
	}
	/* #nosec */
	defer resp.Close()
	t, err := resp.Token()
	if err != nil {
		return err
	}
	_, err = stanza.UnmarshalIQError(resp, t.(xml.StartElement))
	return err
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (r Response) TokenReader() xml.TokenReader {
	return r.IQ.Wrap(xmlstream.Wrap(
		nil,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "command"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "node"}, Value: r.Node},
				{Name: xml.Name{Local: "sessionid"}, Value: r.SID},
				{Name: xml.Name{Local: "status"}, Value: r.Status},
			},
		},
	))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (r Response) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, r.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (r Response) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := r.WriteXML(e)
	return err
}
