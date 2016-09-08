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
	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
)

const (
	bindIQServerGeneratedRP = `<iq id='%s' type='set'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></iq>`
	bindIQClientRequestedRP = `<iq id='%s' type='set'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><resource>%s</resource></bind></iq>`
)

// BindResource is a stream feature that can be used for binding a resource.
func BindResource() StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: ns.Bind, Local: "bind"},
		Necessary:  Authn,
		Prohibited: Ready,
		List: func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error) {
			req = true
			if err = e.EncodeToken(start); err != nil {
				return req, err
			}
			if err = e.EncodeToken(start.End()); err != nil {
				return req, err
			}

			err = e.Flush()
			return req, err
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
			}{}
			return true, nil, d.DecodeElement(&parsed, start)
		},
		Negotiate: func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rwc io.ReadWriteCloser, err error) {
			if (session.state & Received) == Received {
				panic("xmpp: bind not yet implemented")
			}

			conn := session.Conn()

			reqID := internal.RandomID(internal.IDLen)
			if resource := session.config.Origin.Resourcepart(); resource == "" {
				// Send a request for the server to set a resource part.
				_, err = fmt.Fprintf(conn, bindIQServerGeneratedRP, reqID)
			} else {
				// Request the provided resource part.
				_, err = fmt.Fprintf(conn, bindIQClientRequestedRP, reqID, resource)
			}
			if err != nil {
				return mask, nil, err
			}
			tok, err := session.in.d.Token()
			if err != nil {
				return mask, nil, err
			}
			start, ok := tok.(xml.StartElement)
			if !ok {
				return mask, nil, streamerror.BadFormat
			}
			resp := struct {
				IQ
				Bind struct {
					JID *jid.JID `xml:"jid"`
				} `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
				Err StanzaError `xml:"error"`
			}{}
			switch start.Name {
			case xml.Name{Space: ns.Client, Local: "iq"}:
				if err = session.in.d.DecodeElement(&resp, &start); err != nil {
					return mask, nil, err
				}
			default:
				return mask, nil, streamerror.BadFormat
			}

			switch {
			case resp.ID != reqID:
				return mask, nil, streamerror.UndefinedCondition
			case resp.Type == ResultIQ:
				session.origin = resp.Bind.JID
			case resp.Type == ErrorIQ:
				return mask, nil, resp.Err
			default:
				return mask, nil, StanzaError{Condition: BadRequest}
			}
			return Ready, nil, nil
		},
	}
}
