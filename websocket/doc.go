// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package websocket implements a WebSocket transport for XMPP.
package websocket // import "mellium.im/xmpp/websocket"

// Various constants used by this package, provided as a convenience.
const (
	// NS is the XML namespace used by the XMPP subprotocol framing.
	NS = "urn:ietf:params:xml:ns:xmpp-framing"

	// WSProtocol is the protocol string used during the WebSocket handshake.
	WSProtocol = "xmpp"
)
