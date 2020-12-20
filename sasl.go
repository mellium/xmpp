// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/saslerr"
	"mellium.im/xmpp/stream"
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
func SASL(identity, password string, mechanisms ...sasl.Mechanism) StreamFeature {
	if len(mechanisms) == 0 {
		panic("xmpp: Must specify at least 1 SASL mechanism")
	}
	return StreamFeature{
		Name:       xml.Name{Space: ns.SASL, Local: "mechanisms"},
		Necessary:  Secure,
		Prohibited: Authn,
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
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
				List    []string `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanism"`
			}{}
			err := d.DecodeElement(&parsed, start)
			return true, parsed.List, err
		},
		Negotiate: func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, err error) {
			if (session.State() & Received) == Received {
				panic("SASL server not yet implemented")
			}

			c := session.Conn()

			var selected sasl.Mechanism
			// Select a mechanism, preferring the client order.
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

			more, resp, err := client.Step(nil)
			if err != nil {
				return mask, nil, err
			}

			// RFC6120 §6.4.2:
			//     If the initiating entity needs to send a zero-length initial
			//     response, it MUST transmit the response as a single equals sign
			//     character ("="), which indicates that the response is present but
			//     contains no data.
			var encodedResp []byte
			if len(resp) == 0 {
				encodedResp = []byte{'='}
			} else {
				encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
				base64.StdEncoding.Encode(encodedResp, resp)
			}

			// Send <auth/> and the initial payload to start SASL auth.
			if _, err = fmt.Fprintf(c,
				`<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='%s'>%s</auth>`,
				selected.Name, encodedResp,
			); err != nil {
				return mask, nil, err
			}

			r := session.TokenReader()
			defer r.Close()
			d := xml.NewTokenDecoder(r)

			// If we're already done after the first step, decode the <success/> or
			// <failure/> before we exit.
			if !more {
				tok, err := d.Token()
				if err != nil {
					return mask, nil, err
				}
				if t, ok := tok.(xml.StartElement); ok {
					// TODO: Handle the additional data that could be returned if
					// success?
					_, _, err := decodeSASLChallenge(d, t, false)
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
				tok, err := d.Token()
				if err != nil {
					return mask, nil, err
				}
				var challenge []byte
				if t, ok := tok.(xml.StartElement); ok {
					challenge, success, err = decodeSASLChallenge(d, t, true)
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
				_, err = fmt.Fprintf(c, `<response xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>%s</response>`, encodedResp)
				if err != nil {
					return mask, nil, err
				}
			}
			return Authn, c, nil
		},
	}
}

func decodeSASLChallenge(d *xml.Decoder, start xml.StartElement, allowChallenge bool) (challenge []byte, success bool, err error) {
	switch start.Name {
	case xml.Name{Space: ns.SASL, Local: "challenge"}, xml.Name{Space: ns.SASL, Local: "success"}:
		if !allowChallenge && start.Name.Local == "challenge" {
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

		return decodedChallenge, start.Name.Local == "success", nil
	case xml.Name{Space: ns.SASL, Local: "failure"}:
		fail := saslerr.Failure{}
		if err = d.DecodeElement(&fail, &start); err != nil {
			return nil, false, err
		}
		return nil, false, fail
	default:
		return nil, false, stream.UnsupportedStanzaType
	}
}
