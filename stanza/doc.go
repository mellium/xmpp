// Copyright 2017 The Mellium Contributors.
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
// IQ (Info/Query) is a request response mechanism for data that requires a
// response (eg. fetching an avatar or a list of client features).
//
// There are two APIs for creating stanzas in this package, a token based XML
// stream API where the final stanza can be read from an xml.TokenReader, and a
// struct based API that relies on embedding structs in this package into the
// users own types.
// Stanzas created using either API are not guaranteed to be valid or enforce
// specific stanza semantics.
//
// Custom Stanzas
//
// The stanza types in this package aren't very useful by themselves. To
// transmit meaningful data our stanzas must contain a payload.
// To add a payload with the struct based API we use composition to create a new
// struct where the payload is represented by additional fields.
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
//
// For details on marshaling and the use of the xml tag, refer to the
// encoding/xml package.
//
// We could also create a similar stanza with the token stream API:
//
//    // PingIQ returns an xml.TokenReader that outputs a new IQ stanza with a
//    // ping payload.
//    func PingIQ(id string, to jid.JID) xml.TokenReader {
//        start := xml.StartElement{Name: xml.Name{Space: "urn:xmpp:ping", Local: "ping"}}
//        return stanza.IQ{
//            ID: id,
//            To: to,
//            Type: stanza.GetIQ,
//        }.Wrap(start)
//    }
package stanza // import "mellium.im/xmpp/stanza"
