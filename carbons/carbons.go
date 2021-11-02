// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -receiver "h Handler"

// Package carbons implements carbon copying messages to all interested clients.
package carbons // import "mellium.im/xmpp/carbons"

import (
	"context"
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/forward"
	"mellium.im/xmpp/stanza"
)

// Namespaces used by this package, provided as a convenience.
const (
	NS      = `urn:xmpp:carbons:2`
	NSRules = `urn:xmpp:carbons:rules:0`
)

// Enable instructs the server to start carbon copying messages on the given
// session.
func Enable(ctx context.Context, s *xmpp.Session) error {
	return EnableIQ(ctx, s, stanza.IQ{})
}

// EnableIQ is like Enable but it allows you to customize the IQ stanza being
// sent.
// Changing the type of the IQ has no effect.
func EnableIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ) error {
	iq.Type = stanza.SetIQ
	v := struct {
		XMLName xml.Name
	}{}
	err := s.UnmarshalIQ(ctx, iq.Wrap(xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "enable"}},
	)), &v)
	return err
}

// Disable instructs the server to stop carbon copying messages on the given
// session.
func Disable(ctx context.Context, s *xmpp.Session) error {
	return DisableIQ(ctx, s, stanza.IQ{})
}

// DisableIQ is like Disable but it allows you to customize the IQ stanza being
// sent.
// Changing the type of the IQ has no effect.
func DisableIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ) error {
	iq.Type = stanza.SetIQ
	v := struct {
		XMLName xml.Name
	}{}
	err := s.UnmarshalIQ(ctx, iq.Wrap(xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "disable"}},
	)), &v)
	return err
}

type Received struct {
	XMLName xml.Name `xml:"urn:xmpp:carbons:2 received"`
}

func (received Received) Wrap(r xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(
		r,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "received"}},
	)
}

func WrapReceived(received time.Time, r xml.TokenReader) xml.TokenReader {
	return Received{}.Wrap(
		forward.Forwarded{
			Delay: delay.Delay{
				Time: received,
			},
		}.Wrap(r),
	)
}

type Sent struct {
	XMLName xml.Name `xml:"urn:xmpp:carbons:2 sent"`
}

func (sent Sent) Wrap(r xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(
		r,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "sent"}},
	)
}

func WrapSent(received time.Time, r xml.TokenReader) xml.TokenReader {
	return Sent{}.Wrap(
		forward.Forwarded{
			Delay: delay.Delay{
				Time: received,
			},
		}.Wrap(r),
	)
}
