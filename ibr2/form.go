// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"

	"mellium.im/xmpp/form"
)

// Form is a challenge that presents or receives a data form as specified in
// XEP-0004.
func Form(data *form.Data, f func(*form.Data) error) Challenge {
	return Challenge{
		Type: form.NS,
		Send: func(ctx context.Context, e *xml.Encoder) error {
			return e.Encode(data)
		},
		Respond: func(context.Context, *xml.Encoder) error {
			return nil
		},
		Receive: func(ctx context.Context, server bool, d *xml.Decoder, start *xml.StartElement) error {
			return nil
		},
	}
}
