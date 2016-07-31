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
	"net"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
)

// BUG(ssw): STARTTLS feature does not have security layer byte precision.

var (
	ErrTLSUpgradeFailed = errors.New("The underlying connection cannot be upgraded to TLS")
)

// StartTLS returns a new stream feature that can be used for negotiating TLS.
// For StartTLS to work, the underlying connection must support TLS (it must
// implement net.Conn).
func StartTLS(required bool) StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Local: "starttls", Space: ns.StartTLS},
		Prohibited: Secure,
		List: func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error) {
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
			if err = e.EncodeToken(start.End()); err != nil {
				return required, err
			}
			return required, e.Flush()
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
				Required struct {
					XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls required"`
				}
			}{}
			err := d.DecodeElement(&parsed, start)
			return parsed.Required.XMLName.Local == "required" && parsed.Required.XMLName.Space == ns.StartTLS, nil, err
		},
		Negotiate: func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, rwc io.ReadWriteCloser, err error) {
			netconn, ok := conn.Raw().(net.Conn)
			if !ok {
				return mask, nil, ErrTLSUpgradeFailed
			}

			// Fetch or create a TLSConfig to use.
			var tlsconf *tls.Config
			if conn.config.TLSConfig == nil {
				tlsconf = &tls.Config{
					ServerName: conn.LocalAddr().(*jid.JID).Domain().String(),
				}
			} else {
				tlsconf = conn.config.TLSConfig
			}

			if (conn.state & Received) == Received {
				fmt.Fprint(conn, `<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)
				rwc = tls.Server(netconn, tlsconf)
			} else {
				// Select starttls for negotiation.
				fmt.Fprint(conn, `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`)

				// Receive a <proceed/> or <failure/> response from the server.
				t, err := conn.in.d.Token()
				if err != nil {
					return mask, nil, err
				}
				switch tok := t.(type) {
				case xml.StartElement:
					switch {
					case tok.Name.Space != ns.StartTLS:
						return mask, nil, streamerror.UnsupportedStanzaType
					case tok.Name.Local == "proceed":
						// Skip the </proceed> token.
						if err = conn.in.d.Skip(); err != nil {
							return mask, nil, streamerror.InvalidXML
						}
						rwc = tls.Client(netconn, tlsconf)
					case tok.Name.Local == "failure":
						// Skip the </failure> token.
						if err = conn.in.d.Skip(); err != nil {
							err = streamerror.InvalidXML
						}
						// Failure is not an "error", it's expected behavior. Immediately
						// afterwards the server will end the stream. However, if we
						// encounter bad XML while skipping the </failure> token, return
						// that error.
						return mask, nil, err
					default:
						return mask, nil, streamerror.UnsupportedStanzaType
					}
				default:
					return mask, nil, streamerror.RestrictedXML
				}
			}
			mask = Secure
			return
		},
	}
}
