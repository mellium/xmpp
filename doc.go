// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpp provides functionality from the Extensible Messaging and
// Presence Protocol, sometimes known as "Jabber".
//
// It is subdivided into several packages; this package provides functionality
// for establishing an XMPP session, feature negotiation (including an API for
// defining your own stream features), and low-level connection and stream
// manipulation.
// Other important packages include the jid package, which provides an
// implementation of the XMPP address format, and the mux package which provides
// an XMPP handler that can multiplex payloads to other handlers and
// functionality for creating your own multiplexers.
//
// Session Negotiation
//
// To create an XMPP session, most users will want to call DialClientSession or
// DialServerSession to create a client-to-server (c2s) or server-to-server
// (s2s) connection respectively.
// These methods use sane defaults to dial a TCP connection and perform stream
// negotiation.
//
//     session, err := xmpp.DialClientSession(
//         context.TODO(),
//         jid.MustParse("me@example.net"),
//         xmpp.StartTLS(&tls.Config{…}),
//         xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
//         xmpp.BindResource(),
//     )
//
// These functions are a convenience that dial a connection and establish an
// XMPP session in one step.
// If control over DNS or HTTP-based service discovery is desired, the user can
// use the dial package to create a connection and then use one of the other
// session negotiate functions for full control over session initialization.
//
// Aside from the Dial functions there are 6 other functions for establishing a
// session.
// Their names are matched by the regular expression:
//
//     (New|Receive)(Client|Server)?Session
//
// If "New" is present it indicates that the session is from the initiating
// entities perspective while "Receive" indicates the receiving entity.
// If "Client" or "Server" are present they indicate a C2S or S2S connection
// respectively, otherwise they take a Negotiator and its behavior controls or
// detects the type of stream.
//
// This also lets the user create the XMPP session over something other than a
// TCP socket; for example a Unix domain socket or an in-memory pipe.
// It even allows the use of a different session negotiation protocol altogether
// such as the WebSocket subprotocol from the websocket package, or the Jabber
// Component Protocol from the component package.
//
//     conn, err := dial.Client(context.TODO(), "tcp", addr)
//     …
//     session, err := xmpp.NewSession(
//         context.TODO(), addr.Domain(), addr, conn, xmpp.Secure,
//         xmpp.NewNegotiator(xmpp.StreamConfig{
//             Lang: "en",
//             …
//         }),
//     )
//
// The default Negotiator and related functions use a list of StreamFeature's to
// negotiate the state of the session.
// Implementations of the most common features (StartTLS, SASL-based
// authentication, and resource binding) are provided.
// Custom stream features may be created using the StreamFeature struct.
// StreamFeatures defined in this module are safe for concurrent use by multiple
// goroutines and may be created once and then re-used.
//
// Handling Stanzas
//
// Unlike HTTP, the XMPP protocol is asynchronous, meaning that both clients and
// servers can accept and send requests at any time and responses are not
// required or may be received out of order.
// This is accomplished with two XML streams: an input stream and an output
// stream.
// To receive XML on the input stream, Session implements the xml.TokenReader
// interface defined in encoding/xml; this allows session to be wrapped with
// xml.NewTokenDecoder.
// To send XML on the output stream, Session has a TokenEncoder method that
// returns a token encoder that holds a lock on the output stream until it is
// closed.
// The session may also buffer writes and has a Flush method which will write
// any buffered XML to the underlying connection.
//
// Writing individual XML tokens can be tedious and error prone.
// The stanza package contains functions and structs that aid in the
// construction of message, presence and info/query (IQ) elements which have
// special semantics in XMPP and are known as "stanzas".
// These can be sent with the Send, SendElement, SendIQ, and SendIQElement
// methods.
//
//     // Send initial presence to let the server know we want to receive messages.
//     _, err = session.Send(context.TODO(), stanza.WrapPresence(jid.JID{}, stanza.AvailablePresence, nil))
//
// For SendIQ to correctly handle IQ responses, and to make the common case of
// polling for incoming XML on the input stream—and possibly writing to the
// output stream in response—easier, we need a long running goroutine.
// Session includes the Serve method for starting this processing.
//
// Serve provides a Handler with access to the stream but prevents it from
// advancing the stream beyond the current element and always advances the
// stream to the end of the element when the handler returns (even if the
// handler did not consume the entire element).
//
//     err := session.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
//         d := xml.NewTokenDecoder(t)
//
//         // Ignore anything that's not a message.
//         if start.Name.Local != "message" {
//             return nil
//         }
//
//         msg := struct {
//             stanza.Message
//             Body string `xml:"body"`
//         }{}
//         err := d.DecodeElement(&msg, start)
//         …
//         if msg.Body != "" {
//             log.Println("Got message: %q", msg.Body)
//         }
//     }))
//
// It isn't always practical to put all of your logic for handling elements into
// a single function or method, so the mux package contains an XML multiplexer
// that can be used to match incoming payloads against a pattern and delegate
// them to individual handlers.
// Packages that implement extensions to the core XMPP protocol will often
// provide handlers that are compatible with types defined in the mux package,
// and options for registering them with the multiplexer.
//
//
// Be Advised
//
// This API is unstable and subject to change.
package xmpp // import "mellium.im/xmpp"
