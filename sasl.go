// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/sasl"
	"mellium.im/xmpp/internal/saslerr"
	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
)

// BUG(ssw): We can't support server side SASL yet until the SASL library
//           supports it.
//
// BUG(ssw): SASL feature does not have security layer byte precision.

// SASL returns a stream feature for performing authentication using the Simple
// Authentication and Security Layer (SASL) as defined in RFC 4422. It panics if
// no mechanisms are specified. The order in which mechanisms are specified will
// be the prefered order, so stronger mechanisms should be listed first.
func SASL(mechanisms ...sasl.Mechanism) StreamFeature {
	if len(mechanisms) == 0 {
		panic("xmpp: Must specify at least 1 SASL mechanism")
	}
	return StreamFeature{
		Name:       xml.Name{Space: ns.SASL, Local: "mechanisms"},
		Necessary:  Secure,
		Prohibited: Authn,
		List: func(ctx context.Context, conn io.Writer) (req bool, err error) {
			req = true
			_, err = fmt.Fprint(conn, `<mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>`)
			if err != nil {
				return
			}

			for _, m := range mechanisms {
				select {
				case <-ctx.Done():
					return true, ctx.Err()
				default:
				}

				if _, err = fmt.Fprint(conn, `<mechanism>`); err != nil {
					return
				}
				if err = xml.EscapeText(conn, []byte(m.Name)); err != nil {
					return
				}
				if _, err = fmt.Fprint(conn, `</mechanism>`); err != nil {
					return
				}
			}
			_, err = fmt.Fprint(conn, `</mechanisms>`)
			return
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
				List    []string `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanism"`
			}{}
			err := d.DecodeElement(&parsed, start)
			return true, parsed.List, err
		},
		Negotiate: func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, err error) {
			if (conn.state & Received) == Received {
				panic("SASL server not yet implemented")
			} else {
				var selected sasl.Mechanism
				// Select a mechanism, prefering the client order.
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
					return mask, errors.New(`No matching SASL mechanisms found`)
				}

				var client sasl.Negotiator
				// Create the SASL Client
				if tlsconn, ok := conn.rwc.(*tls.Conn); ok {
					client = sasl.NewClient(selected,
						sasl.RemoteMechanisms(data.([]string)...),
						sasl.ConnState(tlsconn.ConnectionState()),
					)
				} else {
					client = sasl.NewClient(selected,
						sasl.RemoteMechanisms(data.([]string)...),
					)
				}

				more, resp, err := client.Step(nil)
				if err != nil {
					return mask, err
				}

				// Send <auth/> and the initial payload to start SASL auth.
				if _, err = fmt.Fprintf(conn,
					`<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl' mechanism='%s'>%s</auth>`,
					selected.Name, resp); err != nil {
					return mask, err
				}

				success := false
				for more {
					select {
					case <-ctx.Done():
						return mask, ctx.Err()
					default:
					}
					tok, err := conn.in.d.Token()
					if err != nil {
						return mask, err
					}
					var challenge []byte
					if t, ok := tok.(xml.StartElement); ok {
						challenge, success, err = decodeSASLChallenge(conn.in.d, t)
						if err != nil {
							return mask, err
						}
					} else {
						return mask, streamerror.BadFormat
					}
					if more, resp, err = client.Step(challenge); err != nil {
						return mask, err
					}
					if !more && success {
						break
					}
					// TODO: What happens if there's more and success (broken server)?
					if _, err = fmt.Fprintf(conn,
						`<response xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>%s</response>`, resp); err != nil {
						return mask, err
					}
				}
				return Authn | StreamRestartRequired, nil
			}
		},
	}
}

func decodeSASLChallenge(d *xml.Decoder, start xml.StartElement) (challenge []byte, success bool, err error) {
	switch start.Name {
	case xml.Name{Space: ns.SASL, Local: "challenge"}, xml.Name{Space: ns.SASL, Local: "success"}:
		challenge := struct {
			Data []byte `xml:",chardata"`
		}{}
		if err = d.DecodeElement(&challenge, &start); err != nil {
			return nil, false, err
		}
		return challenge.Data, start.Name.Local == "success", nil
	case xml.Name{Space: ns.SASL, Local: "failure"}:
		fail := saslerr.Failure{}
		if err = d.DecodeElement(&fail, &start); err != nil {
			return nil, false, err
		}
		return nil, false, fail
	default:
		return nil, false, streamerror.UnsupportedStanzaType
	}
}
