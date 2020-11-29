// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/stream"
)

// Namespaces used by IBR.
const (
	NS = "urn:xmpp:register:0"
)

var (
	errNoChallenge = errors.New("no supported challenges were found")
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

func listFunc(challenges ...Challenge) func(context.Context, xmlstream.TokenWriter, xml.StartElement) (bool, error) {
	return func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error) {
		if err = e.EncodeToken(start); err != nil {
			return req, err
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
				return req, err
			}
			if err = e.EncodeToken(xml.CharData(c.Type)); err != nil {
				return req, err
			}
			if err = e.EncodeToken(challengeStart.End()); err != nil {
				return req, err
			}
			seen[c.Type] = struct{}{}
		}

		err = e.EncodeToken(start.End())
		return req, err
	}
}

func parseFunc(challenges ...Challenge) func(context.Context, *xml.Decoder, *xml.StartElement) (req bool, supported interface{}, err error) {
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

func decodeClientResp(ctx context.Context, r xml.TokenReader, decode func(ctx context.Context, server bool, r xml.TokenReader, start *xml.StartElement) error) (cancel bool, err error) {
	var tok xml.Token
	tok, err = r.Token()
	if err != nil {
		return
	}
	start, ok := tok.(xml.StartElement)
	switch {
	case !ok:
		err = stream.RestrictedXML
		return
	case start.Name.Local == "cancel" && start.Name.Space == NS:
		cancel = true
		return
	case start.Name.Local == "response" && start.Name.Space == NS:
		err = decode(ctx, true, r, &start)
		if err != nil {
			return
		}
	}

	err = stream.BadFormat
	return
}

func negotiateFunc(challenges ...Challenge) func(context.Context, *xmpp.Session, interface{}) (xmpp.SessionState, io.ReadWriter, error) {
	return func(ctx context.Context, session *xmpp.Session, supported interface{}) (xmpp.SessionState, io.ReadWriter, error) {
		server := (session.State() & xmpp.Received) == xmpp.Received
		w := session.TokenWriter()
		defer w.Close()
		r := session.TokenReader()
		defer r.Close()

		if !server && !supported.(bool) {
			// We don't support some of the challenge types advertised by the server.
			// This is not an error, so don't return one; it just means we shouldn't
			// be negotiating this feature.
			return 0, nil, nil
		}

		var tok xml.Token

		if server {
			for _, c := range challenges {
				// Send the challenge.
				start := challengeStart(c.Type)
				err := w.EncodeToken(start)
				if err != nil {
					return 0, nil, err
				}
				err = c.Send(ctx, w)
				if err != nil {
					return 0, nil, err
				}
				err = w.EncodeToken(start.End())
				if err != nil {
					return 0, nil, err
				}
				err = w.Flush()
				if err != nil {
					return 0, nil, err
				}

				// Decode the clients response
				var cancel bool
				cancel, err = decodeClientResp(ctx, r, c.Receive)
				if err != nil || cancel {
					return 0, nil, err
				}
			}
			return 0, nil, nil
		}

		// If we're the client, decode the challenge.
		tok, err := r.Token()
		if err != nil {
			return 0, nil, err
		}
		start, ok := tok.(xml.StartElement)
		switch {
		case !ok:
			return 0, nil, stream.RestrictedXML
		case start.Name.Local != "challenge" || start.Name.Space != NS:
			return 0, nil, stream.BadFormat
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
			return 0, nil, stream.BadFormat
		}

		for _, c := range challenges {
			if c.Type != typ {
				continue
			}

			err = c.Receive(ctx, false, r, &start)
			if err != nil {
				return 0, nil, err
			}

			respStart := xml.StartElement{
				Name: xml.Name{Local: "response"},
			}
			if err = w.EncodeToken(respStart); err != nil {
				return 0, nil, err
			}
			if c.Respond != nil {
				err = c.Respond(ctx, w)
				if err != nil {
					return 0, nil, err
				}
			}
			if err = w.EncodeToken(respStart.End()); err != nil {
				return 0, nil, err
			}

			break
		}
		return 0, nil, nil
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
