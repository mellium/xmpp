// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"
)

// Challenge is an IBR challenge.
type Challenge struct {
	// Type is the type of the challenge as it appears in the server advertised
	// challenges list.
	Type string

	// Send is used by the server to send the challenge to the client.
	Send func(ctx context.Context, e *xml.Encoder) error

	// Respond is used by the client to send a response or reply to the challenge.
	Respond func(context.Context, *xml.Encoder) error

	// Receive is used by the client to receive and decode the server's challenge
	// and by the server to receive and decode the clients response.
	Receive func(ctx context.Context, server bool, d *xml.Decoder, start *xml.StartElement) error
}
