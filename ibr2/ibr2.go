// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package ibr2 implements the Extensible In-Band Registration ProtoXEP.
//
// BE ADVISED: This API is incomplete and is subject to change.
// Core functionality of this package is missing, and the entire package may be
// removed at any time.
package ibr2 // import "mellium.im/xmpp/ibr2"

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmpp"
	"mellium.im/xmpp/streamerror"
)

// Namespaces used by IBR.
const (
	NS = "urn:xmpp:register:0"
)

var (
	errNoChallenge = errors.New("No supported challenges were found")
)

func challengeStart(typ string) xml.StartElement {
	return xml.StartElement{
		Name: xml.Name{
			Space: NS,
			Local: "challenge",
		},
		Attr: []xml.Attr{
			{
				Name:  xml.Name{Local: "type"},
				Value: typ,
			},
		},
	}
}

func listFunc(challenges ...Challenge) func(context.Context, *xml.Encoder, xml.StartElement) (bool, error) {
	return func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error) {
		if err = e.EncodeToken(start); err != nil {
			return
		}

		// List challenges
		seen := make(map[string]struct{})
		for _, c := range challenges {
			if _, ok := seen[c.Type]; ok {
				continue
			}
			challengeStart := xml.StartElement{
				Name: xml.Name{Local: "challenge"},
			}
			if err = e.EncodeToken(challengeStart); err != nil {
				return
			}
			if err = e.EncodeToken(xml.CharData(c.Type)); err != nil {
				return
			}
			if err = e.EncodeToken(challengeStart.End()); err != nil {
				return
			}
			seen[c.Type] = struct{}{}
		}

		if err = e.EncodeToken(start.End()); err != nil {
			return
		}
		return req, e.Flush()
	}
}

func parseFunc(challenges ...Challenge) func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (req bool, supported interface{}, err error) {
	return func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
		// Parse the list of challenge types sent down by the server.
		parsed := struct {
			Challenges []string `xml:"urn:xmpp:register:0 challenge"`
		}{}
		err := d.DecodeElement(&parsed, start)
		if err != nil {
			return false, false, err
		}

		// Dedup the lists of all challenge types supported by us and all challenge
		// types supported by the server.
		m := make(map[string]struct{})
		for _, c := range challenges {
			m[c.Type] = struct{}{}
		}
		for _, c := range parsed.Challenges {
			m[c] = struct{}{}
		}

		// If there are fewer types in the deduped aggregate list than in the
		// challenges we support, then the server list is a subset of the list we
		// support and we're okay to proceed with negotiation.
		return false, len(m) <= len(challenges), nil
	}
}

func decodeClientResp(ctx context.Context, d *xml.Decoder, decode func(ctx context.Context, server bool, d *xml.Decoder, start *xml.StartElement) error) (cancel bool, err error) {
	var tok xml.Token
	tok, err = d.Token()
	if err != nil {
		return
	}
	start, ok := tok.(xml.StartElement)
	switch {
	case !ok:
		err = streamerror.RestrictedXML
		return
	case start.Name.Local == "cancel" && start.Name.Space == NS:
		cancel = true
		return
	case start.Name.Local == "response" && start.Name.Space == NS:
		err = decode(ctx, true, d, &start)
		if err != nil {
			return
		}
	}

	err = streamerror.BadFormat
	return
}

func negotiateFunc(challenges ...Challenge) func(context.Context, *xmpp.Session, interface{}) (xmpp.SessionState, io.ReadWriter, error) {
	return func(ctx context.Context, session *xmpp.Session, supported interface{}) (mask xmpp.SessionState, rw io.ReadWriter, err error) {
		server := (session.State() & xmpp.Received) == xmpp.Received

		if !server && !supported.(bool) {
			// We don't support some of the challenge types advertised by the server.
			// This is not an error, so don't return one; it just means we shouldn't
			// be negotiating this feature.
			return
		}

		var tok xml.Token
		e := session.Encoder()
		d := session.Decoder()

		if server {
			for _, c := range challenges {
				// Send the challenge.
				start := challengeStart(c.Type)
				err = e.EncodeToken(start)
				if err != nil {
					return
				}
				err = c.Send(ctx, e)
				if err != nil {
					return
				}
				err = e.EncodeToken(start.End())
				if err != nil {
					return
				}
				err = e.Flush()
				if err != nil {
					return
				}

				// Decode the clients response
				var cancel bool
				cancel, err = decodeClientResp(ctx, d, c.Receive)
				if err != nil || cancel {
					return
				}
			}
			return
		}

		// If we're the client, decode the challenge.
		tok, err = d.Token()
		if err != nil {
			return
		}
		start, ok := tok.(xml.StartElement)
		switch {
		case !ok:
			err = streamerror.RestrictedXML
			return
		case start.Name.Local != "challenge" || start.Name.Space != NS:
			err = streamerror.BadFormat
			return
		}
		var typ string
		for _, attr := range start.Attr {
			if attr.Name.Local == "type" {
				typ = attr.Value
				break
			}
		}
		// If there was no type attr, an illegal challenge was sent.
		if typ == "" {
			err = streamerror.BadFormat
			return
		}

		for _, c := range challenges {
			if c.Type != typ {
				continue
			}

			err = c.Receive(ctx, false, d, &start)
			if err != nil {
				return
			}

			if c.Respond != nil {
				err = c.Respond(ctx, e)
				if err != nil {
					return
				}
			}

			break
		}
		return
	}
}

// Register returns a new xmpp.StreamFeature that can be used to register a new
// account with the server.
func Register(challenges ...Challenge) xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:       xml.Name{Local: "register", Space: NS},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List:       listFunc(challenges...),
		Parse:      parseFunc(challenges...),
		Negotiate:  negotiateFunc(challenges...),
	}
}

// Recovery returns a new xmpp.StreamFeature that can be used to recover an
// account for which authentication credentials have been lost.
func Recovery(challenges ...Challenge) xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:       xml.Name{Local: "recovery", Space: NS},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List:       listFunc(challenges...),
		Parse:      parseFunc(challenges...),
		Negotiate:  negotiateFunc(challenges...),
	}
}
