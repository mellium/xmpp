// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/sasl"
)

// BUG(ssw): We can't support server side SASL yet until the SASL library
//           supports it.
//
// BUG(ssw): SASL feature does not have security layer byte precision.

// SASL returns a stream feature for performing authentication using the Simple
// Authentication and Security Layer (SASL) as defined in RFC 4422. It panics if
// no mechanisms are specified.
func SASL(mechanisms ...*sasl.Mechanism) *StreamFeature {
	if len(mechanisms) == 0 {
		panic("xmpp: Must specify at least 1 SASL mechanism")
	}
	return &StreamFeature{
		Name:       xml.Name{Space: "urn:ietf:params:xml:ns:xmpp-sasl", Local: "mechanisms"},
		Necessary:  Secure,
		Prohibited: Authn,
		List: func(ctx context.Context, conn io.Writer) (req bool, err error) {
			req = true
			_, err = fmt.Fprint(conn, `<mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>`)
			if err != nil {
				return
			}
			for _, m := range mechanisms {
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
		Negotiate: func(ctx context.Context, conn *Conn, data interface{}) (SessionState, error) {
			if (conn.state & Received) == Received {
				panic("sendMechanisms not yet implemented")
			} else {
				panic("readMechanisms not yet implemented")
			}
			return Authn | StreamRestartRequired, nil
		},
	}
}
