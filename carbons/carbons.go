// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -receiver "h Handler"

// Package carbons implements carbon copying messages to all interested clients.
package carbons // import "mellium.im/xmpp/carbons"

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

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

// WrapReceived wraps the provided token reader (which should be a message
// stanza, but this is not enforced) in a received element.
func WrapReceived(delay delay.Delay, r xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(
		forward.Forwarded{
			Delay: delay,
		}.Wrap(r),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "received"}},
	)
}

// WrapSent wraps the provided token reader (which should be a message stanza,
// but this is not enforced) in a sent element.
func WrapSent(delay delay.Delay, r xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(
		forward.Forwarded{
			Delay: delay,
		}.Wrap(r),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "sent"}},
	)
}

// Unwrap unwraps a carbon copied message, unmarshals the forwarding delay into
// the provided delay, and returns a start element set to either sent or
// received.
// If the provided delay is nil, unmarshaling is skipped.
func Unwrap(del *delay.Delay, r xml.TokenReader) (xml.TokenReader, xml.StartElement, error) {
	token, err := r.Token()
	if err != nil {
		return nil, xml.StartElement{}, err
	}

	se, ok := token.(xml.StartElement)
	if !ok {
		return nil, se, fmt.Errorf("expected a startElement, found %T", token)
	}
	if se.Name.Local != "sent" && se.Name.Local != "received" || se.Name.Space != NS {
		return nil, se, fmt.Errorf("unexpected name for the sent/received element: %+v", se.Name)
	}

	out, err := forward.Unwrap(del, xmlstream.Inner(r))
	return out, se, err
}

// Private is an xmlstream.Transformer that excludes a <message/> from being forwarded to
// other Carbons-enabled resources, by adding a <private/> element and a <no-copy/> hint.
// The first call to consume the output stream will return an error if the input stream
// is not a message element with an appropriate namespace.
func Private(r xml.TokenReader) xml.TokenReader {
	token, err := r.Token()
	if err != nil && err != io.EOF {
		return xmlstream.ReaderFunc(func() (xml.Token, error) {
			return token, err
		})
	}
	se, ok := token.(xml.StartElement)
	if !ok {
		return xmlstream.ReaderFunc(func() (xml.Token, error) {
			return token, fmt.Errorf("expected a startElement, found %T", token)
		})
	}
	if se.Name.Local != "message" || (se.Name.Space != stanza.NSClient && se.Name.Space != stanza.NSServer) {
		return xmlstream.ReaderFunc(func() (xml.Token, error) {
			return token, fmt.Errorf("unexpected name for the message element: %+v", se.Name)
		})
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Inner(r),
			xmlstream.Wrap(nil, xml.StartElement{
				Name: xml.Name{Space: NS, Local: "private"},
			}),
			xmlstream.Wrap(nil, xml.StartElement{
				Name: xml.Name{Space: "urn:xmpp:hints", Local: "no-copy"},
			}),
		),
		se,
	)
}
