// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package stanza contains functionality for dealing with XMPP stanzas and
// stanza level errors.
//
// Stanzas (Message, Presence, and IQ) are the basic building blocks of an XMPP
// stream.
// Messages are used to send data that is fire-and-forget such as chat messages.
// Presence is a publish-subscribe mechanism and is used to broadcast
// availability on the network (sometimes called "status" in chat, eg.  online,
// offline, or away).
// IQ (Info-Query) is a request response mechanism for data that requires a
// response (eg. fetching an avatar or a list of client features).
//
// Stanzas created using the structs in this package are not guaranteed to be
// valid or enforce specific stanza semantics.
// For instance, using this package you could create an IQ without a unique ID,
// which is illegal in XMPP.
// Packages that require correct stanza semantics, such as the `mellium.im/xmpp`
// package, are expected to enforce stanza semantics when encoding stanzas to a
// stream.
//
// Custom Stanzas
//
// The stanza types in this package aren't very useful by themselves. To
// transmit meaningful data our stanzas must contain a payload.
// To add a payload we use composition to create a new struct that contains the
// payload as additional fields.
// For example, XEP-0199: XMPP Ping defines an IQ stanza with a payload named
// "ping" qualified by the "urn:xmpp:ping" namespace.
// To implement this in our own code we might create a Ping struct similar to
// the following:
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
