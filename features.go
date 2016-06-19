// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"golang.org/x/net/context"
)

// A StreamFeature represents a feature that may be selected during stream
// negotiation.
type StreamFeature struct {
	// A function that will take over the session temporarily while negotiating
	// the feature. If StreamRestart is true, the stream will be restarted
	// automatically if Handler does not return an error. SessionState represents
	// the state bits that should be flipped on successful negotiation of the
	// feature. For instance, if this feature upgrades the connection to a
	// TLS connection and performs mutual TLS authentication to log in the user
	// this would be set to Authn|Secure|StreamRestartRequired, but if it does not
	// authenticate the connection it would return Secure|StreamRestartRequired.
	Handler func(ctx context.Context, conn *Conn) (SessionState, error)

	// The XML name of the feature in the <stream:feature/> list.
	Name xml.Name

	// True if negotiating the feature is required.
	Required bool
}
