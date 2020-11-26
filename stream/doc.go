// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package stream implements stream level functionality.
//
// The various stream errors defined by RFC 6120 § 4.9 are included as a
// convenience, but most people will want to use the facilities of the
// mellium.im/xmpp package and not create stream errors directly.
package stream // import "mellium.im/xmpp/stream"

// Namespaces used by XMPP streams and stream errors, provided as a convenience.
const (
	NS      = "http://etherx.jabber.org/streams"
	ErrorNS = "urn:ietf:params:xml:ns:xmpp-streams"
)
