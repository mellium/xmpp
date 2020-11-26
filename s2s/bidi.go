// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package s2s

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
)

// Namespaces used in this package, provided as a convenience.
const (
	// NSBidi is the namespace used for negotiating bidirectional S2S connections.
	NSBidi = "urn:xmpp:bidi"

	// NSBidiFeature is the namespace used for advertising Bidi support.
	NSBidiFeature = "urn:xmpp:features:bidi"
)

// Bidi returns a stream feature for indicating support for bidirectional
// server-to-server connections (ie. s2s connections over a single bidirectional
// TCP connection instead of over two unidirectional TCP connections).
//
// The feature itself is just informational, servers using this feature will
// need to check if it was negotiated and handle their connections
// appropriately.
func Bidi() xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:       xml.Name{Space: NSBidiFeature, Local: "bidi"},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List: func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (bool, error) {
			if err := e.EncodeToken(start); err != nil {
				return false, err
			}
			return false, e.EncodeToken(start.End())
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:xmpp:features:bidi bidi"`
			}{}
			return false, nil, d.DecodeElement(&parsed, start)
		},
		Negotiate: func(ctx context.Context, session *xmpp.Session, data interface{}) (xmpp.SessionState, io.ReadWriter, error) {
			if (session.State() & xmpp.Received) == xmpp.Received {
				// The BIDI feature is just informational at this point, no need to
				// respond if we're a server.
				return 0, nil, nil
			}

			w := session.TokenWriter()
			defer w.Close()

			start := xml.StartElement{
				Name: xml.Name{Space: NSBidi, Local: "bidi"},
			}
			err := w.EncodeToken(start)
			if err != nil {
				return 0, nil, err
			}
			return 0, nil, w.EncodeToken(start.End())
		},
	}
}
