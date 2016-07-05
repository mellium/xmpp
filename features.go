// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
)

// A StreamFeature represents a feature that may be selected during stream
// negotiation. Features may be stateful and the same feature should not be used
// on multiple connections or from multiple goroutines.
type StreamFeature struct {
	// A function that will take over the session temporarily while negotiating
	// the feature. If StreamRestart is true, the stream will be restarted
	// automatically if Handler does not return an error. The "mask" SessionState
	// represents the state bits that should be flipped on successful negotiation
	// of the feature. For instance, if this feature upgrades the connection to a
	// TLS connection and performs mutual TLS authentication to log in the user
	// this would be set to Authn|Secure|StreamRestartRequired, but if it does not
	// authenticate the connection it would return Secure|StreamRestartRequired.
	Handler func(ctx context.Context, conn *Conn) (mask SessionState, err error)

	// The XML name of the feature in the <stream:feature/> list.
	Name xml.Name

	// True if negotiating the feature is required.
	Required bool

	// Bits that are required before this feature is advertised. For instance, if
	// this feature should only be advertised after the user is authenticated we
	// might set this to "Authn" or if it should be advertised only after the
	// feature is authenticated and encrypted we might set this to "Authn|Secure".
	Necessary SessionState

	// Bits that must be off for this feature to be advertised. For instance, if
	// this feature should only be advertised before the connection is
	// authenticated (eg. if the feature performs authentication itself), we might
	// set this to "Authn".
	Prohibited SessionState
}

func (c *Conn) negotiateFeatures(ctx context.Context) error {
	if (c.state & Received) == Received {
		// sendFeatures
	} else {
		// expectFeatures
	}
	return nil
}
