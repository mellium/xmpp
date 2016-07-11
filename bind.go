// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
)

// TODO: Strictly speaking, there's no reason for BindResource to be a function.
//       I have a vague feeling that it should still be one though and need to
//       figure out why…

// BindResource is a stream feature that can be used for binding a resource.
func BindResource() StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: NSBind, Local: "bind"},
		Necessary:  Authn,
		Prohibited: Bind | Ready,
		List: func(ctx context.Context, w io.Writer) (bool, error) {
			_, err := fmt.Fprintf(w, `<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>`)
			return true, err
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
			}{}
			return true, nil, d.DecodeElement(&parsed, start)
		},
		Negotiate: func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, err error) {
			if (conn.state & Received) == Received {
				panic("xmpp: bind not yet implemented")
			} else {
				resource := conn.config.Origin.Resourcepart()
				if resource == "" {
					// Send a request for the server to set a resource part.
				} else {
					// Request the provided resource part.
				}
				panic("bind negotiation: Not yet implemented")
			}
		},
	}
}
