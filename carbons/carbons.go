// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package carbons implements carbon copying messages to all interested clients.
package carbons // import "mellium.im/xmpp/carbons"

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
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

/*
Write a wrapper that disables carbons for a particular message by inserting a
<private xmlns='urn:xmpp:carbons:2'/> element similar to how receipts.Request works.
 */


//DisableCarbon is an xmlstream.Transformer that inserts <private xmlns='urn:xmpp:carbons:2'/>
//into an individual message, which in turn disables carbons for that message

func DisableCarbon(r xml.TokenReader) xml.TokenReader{
	var(
		disabled bool
		inner xml.TokenReader
	)
	return xmlstream.ReaderFunc(func() (xml.Token, error){
		start:
			if inner != nil{
				token, err := inner.Token()
				if err == io.EOF {
					inner = nil
					err = nil
				}

				return token, err
			}

			token, err := r.Token()
			switch err {
			case io.EOF:
				if token == nil{
					return nil, err
				}
				err = nil
			case nil:
			default:
				return token, err
			}

			switch t := token.(type) {
			case xml.StartElement:
				switch{
				case t.Name.Local
				}
		
			}
		
	})
}
