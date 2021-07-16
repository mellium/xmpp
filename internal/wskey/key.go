// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package wskey is a context key used by negotiators.
//
// We are doing exactly what the context package tells us not to do and using it
// to pass optional arguments to the function (in this case, whether to use the
// WebSocket subprotocol or not).
// A better way to do this would be to move the negotiator to an
// internal/negotiator package and have xmpp and websocket both import and use
// that.
// Unfortunately, that would cause import loops (because the negotiator function
// takes an xmpp.Session, so the internal/negotiator package would also need to
// import the xmpp package).
// We could also copy/pate the entire implementation into websocket, but this is
// a maintainability nightmare.
//
// Having a secret internal API may not be ideal, but it does let us get away
// with a nice surface API without any real drawbacks other than an extra tiny
// internal package to house this key.
package wskey // import "mellium.im/xmpp/internal/wskey"

// Key is an internal type used as a context key by the xmpp and websocket
// packages.
// If it is provided on a context to xmpp.NewNegotiator, the WebSocket
// subprotocol is used instead of the normal XMPP protocol.
type Key struct{}
