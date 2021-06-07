// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -vars=FeatureEscaping:`jid\20escaping`

// Package jid implements the XMPP address format.
//
// XMPP addresses, more often called "JID's" (Jabber ID's) for historical
// reasons, comprise three parts:
// The localpart represents a specific user account, the domainpart is the
// domain, host name, or IP address of a server hosting the account, and the
// resourcepart which represents a specific client connected to an account (eg.
// the users phone or a web browser).
// Only the domainpart is required, and together they are formatted like an
// email with the resourcepart appended after a forward slash.
// For example, the following are all valid JIDs:
//
//     shakespeare@example.net
//     shakespeare@example.net/phone-b5c93ded
//     example.net
//
// The first represents the account "shakespeare" on the service "example.net",
// the second represents a specific phone connected to that account, and the
// third represents the server running the service at example.net.
// This means that clients connected to the XMPP network are individually and
// globally addressable.
//
// The jid package also implements the escaping mechanism defined in XEP-0106:
// JID Escaping.
// This can be used to expand the supported characters in the username of a JID.
//
// Be advised: This API is still unstable and is subject to change.
package jid // import "mellium.im/xmpp/jid"
