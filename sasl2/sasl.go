// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package sasl2 is an experimental implementation of XEP-0388: Extensible SASL
// Profile.
//
// BE ADVISED: This API is incomplete and is subject to change.
// Core functionality of this package is missing, and the entire package may be
// removed at any time.
package sasl2 // import "mellium.im/xmpp/sasl2"

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/saslerr"
	"mellium.im/xmpp/stream"
)

// TODO(ssw): Support caching mechanisms on the feature and pipelining the
// selection.

// Namespaces used by SASL2.
const (
	NS = "urn:xmpp:sasl:0"
)

// SASL returns a stream feature for performing authentication using the Simple
// Authentication and Security Layer (SASL) as defined in RFC 4422.
// It panics if no mechanisms are specified.
// The order in which mechanisms are specified will be the preferred order, so
// stronger mechanisms should be listed first.
//
// Identity is used when a user wants to act on behalf of another user.
// For instance, an admin might want to log in as another user to help them
// troubleshoot an issue.
// Normally it is left blank and the localpart of the Origin JID is used.
func SASL(identity, password string, mechanisms ...sasl.Mechanism) xmpp.StreamFeature {
	if len(mechanisms) == 0 {
		panic("sasl2: Must specify at least 1 mechanism")
	}

	return xmpp.StreamFeature{
		Name:       xml.Name{Space: NS, Local: "mechanisms"},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List: func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error) {
			req = true
			if err = e.EncodeToken(start); err != nil {
				return
			}

			startMechanism := xml.StartElement{Name: xml.Name{Space: "", Local: "mechanism"}}
			for _, m := range mechanisms {
				select {
				case <-ctx.Done():
					return true, ctx.Err()
				default:
				}

				if err = e.EncodeToken(startMechanism); err != nil {
					return
				}
				if err = e.EncodeToken(xml.CharData(m.Name)); err != nil {
					return
				}
				if err = e.EncodeToken(startMechanism.End()); err != nil {
					return
				}
			}
			return req, e.EncodeToken(start.End())
		},
		Parse: func(ctx context.Context, r xml.TokenReader, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:xmpp:sasl:0 mechanisms"`
				List    []string `xml:"urn:xmpp:sasl:0 mechanism"`
			}{}
			err := xml.NewTokenDecoder(r).DecodeElement(&parsed, start)
			return true, parsed.List, err
		},
		Negotiate: func(ctx context.Context, session *xmpp.Session, data interface{}) (mask xmpp.SessionState, rw io.ReadWriter, err error) {
			if (session.State() & xmpp.Received) == xmpp.Received {
				panic("SASL server not yet implemented")
			}

			conn := session.Conn()

			// Select a mechanism, preferring the client order.
			var selected sasl.Mechanism
		selectmechanism:
			for _, m := range mechanisms {
				for _, name := range data.([]string) {
					if name == m.Name {
						selected = m
						break selectmechanism
					}
				}
			}
			// No matching mechanism found…
			if selected.Name == "" {
				return mask, nil, errors.New(`No matching SASL mechanisms found`)
			}

			// Create a new SASL client and give it access to credentials, other
			// mechanisms advertised by the server, and the TLS session state if
			// possible (for SCRAM-PLUS mechanisms).
			opts := []sasl.Option{
				sasl.Credentials(func() ([]byte, []byte, []byte) {
					return []byte(session.LocalAddr().Localpart()), []byte(password), []byte(identity)
				}),
				sasl.RemoteMechanisms(data.([]string)...),
			}

			if connState := session.ConnectionState(); connState.Version != 0 {
				opts = append(opts, sasl.TLSState(connState))
			}

			client := sasl.NewClient(selected, opts...)

			// Calculate the initial response
			more, resp, err := client.Step(nil)
			if err != nil {
				return mask, nil, err
			}

			// XEP-0388 §2.2:
			//     In order to explicitly transmit a zero-length SASL challenge or
			//     response, the sending party sends a single equals sign character
			//     ("=").
			var encodedResp []byte
			if len(resp) == 0 {
				encodedResp = []byte{'='}
			} else {
				encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
				base64.StdEncoding.Encode(encodedResp, resp)
			}

			// TODO: Printf'ing is probably a bad idea. Encode the tokens properly.
			// Send <auth/> and the initial payload to start SASL auth.
			if _, err = fmt.Fprintf(conn,
				`<authenticate xmlns='%s' mechanism='%s'><initial-response>%s</initial-response></authenticate>`,
				NS, selected.Name, encodedResp,
			); err != nil {
				return mask, nil, err
			}

			r := session.TokenReader()
			defer r.Close()

			// If we're already done after the first step, decode the <success/> or
			// <failure/> before we exit.
			if !more {
				tok, err := r.Token()
				if err != nil {
					return mask, nil, err
				}
				if t, ok := tok.(xml.StartElement); ok {
					// TODO: Handle the additional data that could be returned if
					// success?
					_, _, err := decodeSASLChallenge(r, t, false)
					if err != nil {
						return mask, nil, err
					}
				} else {
					return mask, nil, stream.BadFormat
				}
			}

			success := false
			for more {
				select {
				case <-ctx.Done():
					return mask, nil, ctx.Err()
				default:
				}
				tok, err := r.Token()
				if err != nil {
					return mask, nil, err
				}
				var challenge []byte
				if t, ok := tok.(xml.StartElement); ok {
					challenge, success, err = decodeSASLChallenge(r, t, true)
					if err != nil {
						return mask, nil, err
					}
				} else {
					return mask, nil, stream.BadFormat
				}
				if more, resp, err = client.Step(challenge); err != nil {
					return mask, nil, err
				}
				if !more && success {
					// We're done with SASL and we're successful
					break
				}

				var encodedResp []byte
				if len(resp) == 0 {
					encodedResp = []byte{'='}
				} else {
					encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
					base64.StdEncoding.Encode(encodedResp, resp)
				}

				// TODO: What happens if there's more and success (broken server)?
				if _, err = fmt.Fprintf(conn,
					`<response xmlns='urn:xmpp:sasl:0'>%s</response>`, encodedResp); err != nil {
					return mask, nil, err
				}
			}
			return xmpp.Authn, conn, nil
		},
	}
}

func decodeSASLChallenge(r xml.TokenReader, start xml.StartElement, allowChallenge bool) (challenge []byte, success bool, err error) {
	d := xml.NewTokenDecoder(r)
	switch start.Name {
	case xml.Name{Space: NS, Local: "challenge"}:
		if !allowChallenge {
			return nil, false, stream.UnsupportedStanzaType
		}
		challenge := struct {
			Data []byte `xml:",chardata"`
		}{}
		if err = d.DecodeElement(&challenge, &start); err != nil {
			return nil, false, err
		}

		decodedChallenge := make([]byte, base64.StdEncoding.DecodedLen(len(challenge.Data)))
		n, err := base64.StdEncoding.Decode(decodedChallenge, challenge.Data)
		if err != nil {
			return nil, false, err
		}
		decodedChallenge = decodedChallenge[:n]

		return decodedChallenge, false, nil
	case xml.Name{Space: NS, Local: "success"}:
		success := struct {
			XMLName xml.Name `xml:"urn:xmpp:sasl:0 success"`
			Data    []byte   `xml:"success-data"`
		}{}
		if err = d.DecodeElement(&challenge, &start); err != nil {
			return nil, true, err
		}

		decodedChallenge := make([]byte, base64.StdEncoding.DecodedLen(len(success.Data)))
		n, err := base64.StdEncoding.Decode(decodedChallenge, success.Data)
		if err != nil {
			return nil, false, err
		}
		decodedChallenge = decodedChallenge[:n]

		return decodedChallenge, true, nil
	case xml.Name{Space: NS, Local: "failure"}:
		fail := saslerr.Failure{}
		if err = d.DecodeElement(&fail, &start); err != nil {
			return nil, false, err
		}
		return nil, false, fail
	default:
		return nil, false, stream.UnsupportedStanzaType
	}
}
