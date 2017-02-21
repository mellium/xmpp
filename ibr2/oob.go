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
// The returned OOB data is ignored for clients.
// For servers, f is also called.
func OOB(f func(*oob.Data) (*oob.Data, error)) Challenge {
	return Challenge{
		Type: oob.NS,
		Send: func(ctx context.Context, e *xml.Encoder) error {
			return nil
		},
	}
}
