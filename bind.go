// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
)

const (
	bindIQServerGeneratedRP = `<iq id='%s' type='set'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></iq>`
	bindIQClientRequestedRP = `<iq id='%s' type='set'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><resource>%s</resource></bind></iq>`
)

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
				reqID := internal.RandomID(internal.IDLen)
				if resource := conn.config.Origin.Resourcepart(); resource == "" {
					// Send a request for the server to set a resource part.
					_, err = fmt.Fprintf(conn, bindIQServerGeneratedRP, reqID)
				} else {
					// Request the provided resource part.
					_, err = fmt.Fprintf(conn, bindIQClientRequestedRP, reqID, resource)
				}
				if err != nil {
					return mask, err
				}
				tok, err := conn.in.d.Token()
				if err != nil {
					return mask, err
				}
				start, ok := tok.(xml.StartElement)
				if !ok {
					return mask, BadFormat
				}
				resp := struct {
					IQ
					Bind struct {
						JID *jid.JID `xml:"jid"`
					} `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
				}{}
				switch start.Name {
				case xml.Name{Space: NSClient, Local: "iq"}:
					if err = conn.in.d.DecodeElement(&resp, &start); err != nil {
						return mask, err
					}
				default:
					return mask, BadFormat
				}

				switch {
				case resp.ID != reqID:
					// TODO: Do we actually care about this? Should this be a stanza error
					// instead?
					return mask, UndefinedCondition
				case resp.Type == ResultIQ:
					conn.origin = resp.Bind.JID
				case resp.Type == ErrorIQ:
					panic("Bind error processing not yet implemented")
				default:
					// bad-request
					panic("Bind invalid request processing not yet implemented")
				}
				return Bind, nil
			}
		},
	}
}
