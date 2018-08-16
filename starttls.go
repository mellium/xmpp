// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/stream"
)

// BUG(ssw): STARTTLS feature does not have security layer byte precision.

// StartTLS returns a new stream feature that can be used for negotiating TLS.
func StartTLS(required bool, cfg *tls.Config) StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Local: "starttls", Space: ns.StartTLS},
		Prohibited: Secure,
		List: func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error) {
			if err = e.EncodeToken(start); err != nil {
				return required, err
			}
			if required {
				startRequired := xml.StartElement{Name: xml.Name{Space: "", Local: "required"}}
				if err = e.EncodeToken(startRequired); err != nil {
					return required, err
				}
				if err = e.EncodeToken(startRequired.End()); err != nil {
					return required, err
				}
			}
			return required, e.EncodeToken(start.End())
		},
		Parse: func(ctx context.Context, r xml.TokenReader, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
				Required struct {
					XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls required"`
				}
			}{}
			err := xml.NewTokenDecoder(r).DecodeElement(&parsed, start)
			return parsed.Required.XMLName.Local == "required" && parsed.Required.XMLName.Space == ns.StartTLS, nil, err
		},
		Negotiate: func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, err error) {
			conn := session.Conn()
			state := session.State()
			d := xml.NewTokenDecoder(session)

			// If no TLSConfig was specified, use a default config.
			if cfg == nil {
				cfg = &tls.Config{
					ServerName: session.LocalAddr().Domain().String(),
				}
			}

			if (state & Received) == Received {
				fmt.Fprint(conn, `<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)
				rw = tls.Server(conn, cfg)
			} else {
				// Select starttls for negotiation.
				fmt.Fprint(conn, `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)

				// Receive a <proceed/> or <failure/> response from the server.
				t, err := d.Token()
				if err != nil {
					return mask, nil, err
				}
				switch tok := t.(type) {
				case xml.StartElement:
					switch {
					case tok.Name.Space != ns.StartTLS:
						return mask, nil, stream.UnsupportedStanzaType
					case tok.Name.Local == "proceed":
						// Skip the </proceed> token.
						if err = d.Skip(); err != nil {
							return mask, nil, stream.InvalidXML
						}
						rw = tls.Client(conn, cfg)
					case tok.Name.Local == "failure":
						// Skip the </failure> token.
						if err = d.Skip(); err != nil {
							err = stream.InvalidXML
						}
						// Failure is not an "error", it's expected behavior. Immediately
						// afterwards the server will end the stream. However, if we
						// encounter bad XML while skipping the </failure> token, return
						// that error.
						return mask, nil, err
					default:
						return mask, nil, stream.UnsupportedStanzaType
					}
				default:
					return mask, nil, stream.RestrictedXML
				}
			}
			mask = Secure
			return
		},
	}
}
