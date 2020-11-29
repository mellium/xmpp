// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/oob"
)

// OOB is a challenge that must be completed out of band using a URI provided by
// XEP-0066: Out of Band Data.
// If you are a client, f will be called and passed the parsed OOB data.
// If f returns an error, the client considers the negotiation a failure.
// For servers, the provided data is encoded and sent as part of the challenge
// (f is not used).
func OOB(data *oob.Data, f func(*oob.Data) error) Challenge {
	return Challenge{
		Type: oob.NS,
		Send: func(ctx context.Context, w xmlstream.TokenWriter) error {
			_, err := data.WriteXML(w)
			return err
		},
		Receive: func(ctx context.Context, server bool, r xml.TokenReader, start *xml.StartElement) error {
			// The server does not receive a reply for this mechanism.
			if server {
				return nil
			}

			oob := &oob.Data{}
			err := xml.NewTokenDecoder(r).Decode(oob)
			if err != nil {
				return err
			}

			return f(oob)
		},
	}
}
