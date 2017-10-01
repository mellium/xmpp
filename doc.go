// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpp provides functionality from the Extensible Messaging and
// Presence Protocol, formerly known as "Jabber".
//
// It is subdivided into several packages; this package provides functionality
// for establishing an XMPP session, feature negotiation (including an API for
// defining your own stream features), and low-level connection and stream
// manipulation.
// The jid package provides an implementation of the XMPP address format defined
// in RFC 7622.
//
// Be advised: This API is still unstable and is subject to change.
package xmpp // import "mellium.im/xmpp"
