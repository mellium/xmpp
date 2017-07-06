// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package stanza contains functionality for dealing with XMPP stanzas and
// stanza level errors.
//
// Stanzas (Message, Presence, and IQ) are the "primitives" of XMPP. Messages
// are used to send data that is fire-and-forget such as chat messages, Presence
// is used as a general broadcast and publish-subscribe mechanism and is used to
// broadcast availability on the network (sometimes called "status" in chat, eg.
// online, offline, or away), and IQ (Info-Query) is used as a request response
// mechanism for data that requires a response (eg. fetching an avatar or a list
// of client features).
package stanza // import "mellium.im/xmpp/stanza"
