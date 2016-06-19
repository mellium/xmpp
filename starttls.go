// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"golang.org/x/net/context"
)

// StartTLS returns a new stream feature that can be used for negotiating TLS.
func StartTLS(required bool) StreamFeature {
	return StreamFeature{
		Handler: func(ctx context.Context, conn *Conn) (state SessionState, err error) {
			state = Secure | StreamRestartRequired
			return
		},
		Name:     xml.Name{Local: "starttls", Space: "urn:ietf:params:xml:ns:xmpp-tls"},
		Required: true,
	}
}
