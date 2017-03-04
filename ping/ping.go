// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package ping implements XEP-0199: XMPP Ping.
package ping

import (
	"mellium.im/xmpp"
)

// BUG(ssw): This package does not currently provide a means of registering a
//           disco#info feature or a response handler.

const ns = `urn:xmpp:ping`

type Ping struct {
	xmpp.IQ

	Ping struct{} `xml:"urn:xmpp:ping ping"`
}
