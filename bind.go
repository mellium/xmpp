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

// BindResource returns a stream feature that can be used for binding a
// resource. The provided resource may (and probably "should") be empty, and is
// merely a suggestion; the server may return a completely different resource to
// prevent conflicts. If BindResource is used on a server connection, the
// resource argument is ignored.
func BindResource(resource string) *StreamFeature {
	return &StreamFeature{
		Name:       xml.Name{Space: NSBind, Local: "bind"},
		Necessary:  Secure,
		Prohibited: Authn,
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
			panic("Not yet implemented")
		},
	}
}
