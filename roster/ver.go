// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package roster

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
)

// Versioning returns a stream feature that advertises roster versioning
// support.
//
// Actually attempting to negotiate the feature does nothing as it is meant to
// be informational only.
func Versioning() xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:      xml.Name{Space: NSFeatures, Local: "ver"},
		Necessary: xmpp.Secure,
		List: func(_ context.Context, e xmlstream.TokenWriter, start xml.StartElement) (bool, error) {
			err := e.EncodeToken(start)
			if err != nil {
				return true, err
			}
			return true, e.EncodeToken(start.End())
		},
		Parse: func(_ context.Context, d *xml.Decoder, _ *xml.StartElement) (bool, interface{}, error) {
			return false, nil, d.Skip()
		},
		Negotiate: func(context.Context, *xmpp.Session, interface{}) (xmpp.SessionState, io.ReadWriter, error) {
			return 0, nil, nil
		},
	}
}
