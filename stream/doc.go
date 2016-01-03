// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// The stream package manages XML streams.
//
// RFC 6120: §4.1 defines an XML stream as follows:
//
//     An XML stream is a container for the exchange of XML elements between any
//     two entities over a network. The start of an XML stream is denoted
//     unambiguously by an opening "stream header" (i.e., an XML <stream> tag
//     with appropriate attributes and namespace declarations), while the end of
//     the XML stream is denoted unambiguously by a closing XML </stream> tag.
//     During the life of the stream, the entity that initiated it can send an
//     unbounded number of XML elements over the stream, either elements used to
//     negotiate the stream (e.g., to complete TLS negotiation (Section 5) or
//     SASL negotiation (Section 6)) or XML stanzas.  The "initial stream" is
//     negotiated from the initiating entity (typically a client or server) to
//     the receiving entity (typically a server), and can be seen as
//     corresponding to the initiating entity's "connection to" or "session
//     with" the receiving entity.  The initial stream enables unidirectional
//     communication from the initiating entity to the receiving entity; in
//     order to enable exchange of stanzas from the receiving entity to the
//     initiating entity, the receiving entity MUST negotiate a stream in the
//     opposite direction (the "response stream").
//
// Be advised: This API is still unstable and is subject to change.
package stream // import "bitbucket.org/mellium/xmpp/stream"
