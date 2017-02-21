// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"

	"mellium.im/xmpp/oob"
)

// OOB is a challenge that must be completed out of band using a URI provided by
// XEP-0066: Out of Band Data.
// If you are a client, f will be called and passed the parsed OOB data.
// If f returns an error, the client considers the negotiation a failure.
// For servers, the provided data is encoded and sent as part of the challenge.
func OOB(data *oob.Data, f func(*oob.Data) error) Challenge {
	return Challenge{
		Type: oob.NS,
		Send: func(ctx context.Context, e *xml.Encoder) error {
			return e.Encode(data)
		},
		Receive: func(ctx context.Context, server bool, d *xml.Decoder, start *xml.StartElement) error {
			// The server does not receive a reply for this mechanism.
			if server {
				return nil
			}

			oob := &oob.Data{}
			err := d.Decode(oob)
			if err != nil {
				return err
			}

			return f(oob)
		},
	}
}
