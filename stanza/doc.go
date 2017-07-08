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
//
//
// Custom Stanzas
//
// The stanza types in this package aren't very useful by themselves. To
// transmit meaningful data we need to add a payload. To add a payload we use
// composition to create a new struct that contains the payload as additional
// fields. For example, XEP-0199: XMPP Ping defines an IQ stanza with a payload
// named "ping" qualified by the "urn:xmpp:ping" namespace. To implement this in
// our own code we might create a Ping struct similar to the following:
//
//
//    // PingIQ is an IQ stanza with an XEP-0199: XMPP Ping payload.
//    type PingIQ struct {
//        stanza.IQ
//
//        Ping struct{} `xml:"urn:xmpp:ping ping"`
//    }
//
// For details on marshaling and the use of the xml tag, refer to the
// encoding/xml package.
package stanza // import "mellium.im/xmpp/stanza"
