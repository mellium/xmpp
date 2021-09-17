// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpp provides functionality from the Extensible Messaging and
// Presence Protocol, sometimes known as "Jabber".
//
// This module is subdivided into several packages.
// This package provides functionality for establishing an XMPP session, feature
// negotiation (including an API for defining your own stream features), and
// handling events.
// Other important packages include the jid package, which provides an
// implementation of the XMPP address format, the mux package which provides an
// XMPP handler that can multiplex payloads to other handlers and functionality
// for creating your own multiplexers, and the stanza package which provides
// functionality for transmitting XMPP primitives and errors.
//
// Session Negotiation
//
// There are 9 functions for establishing an XMPP session.
// Their names are matched by the regular expression:
//
//     (New|Receive|Dial)(Client|Server)?Session
//
// If "Dial" is present it means the function uses sane defaults to dial a TCP
// connection before negotiating an XMPP session on it.
// Most users will want to call DialClientSession or DialServerSession to create
// a client-to-server (c2s) or server-to-server (s2s) connection respectively.
// These methods are the most convenient way to quickly start a connection.
//
//     session, err := xmpp.DialClientSession(
//         context.TODO(),
//         jid.MustParse("me@example.net"),
//         xmpp.StartTLS(&tls.Config{…}),
//         xmpp.SASL("", pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
//         xmpp.BindResource(),
//     )
//
// If control over DNS or HTTP-based service discovery is desired, the user can
// use the dial package to create a connection and then use one of the other
// session negotiation functions for full control over session initialization.
//
// If "New" or "Dial" is present in the function name it indicates that the
// session is from the initiating entities perspective while "Receive" indicates
// the receiving entity.
// If "Client" or "Server" are present they indicate a C2S or S2S connection
// respectively, otherwise the function takes a Negotiator and an initial
// session state to determine the type of session to create.
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
//         xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) {
//             return xmpp.StreamConfig{
//                 Lang: "en",
//                 …
//             },
//         }),
//     )
//
// The default Negotiator and related functions use a list of StreamFeature's to
// negotiate the state of the session.
// Implementations of the most commonly used features (StartTLS, SASL-based
// authentication, and resource binding) are provided.
// Custom stream features may be created using the StreamFeature struct.
// StreamFeatures defined in this module are safe for concurrent use by multiple
// goroutines and may be created once and then re-used.
//
// Handling Stanzas
//
// Unlike HTTP, the XMPP protocol is asynchronous, meaning that both clients and
// servers can accept and send requests at any time and responses are not always
// required or may be received out of order.
// This is accomplished with two XML streams: an input stream and an output
// stream.
// To receive XML on the input stream, Session provides the Serve method which
// takes a handler that has the ability to read incoming XML.
// If the full stream should be read it also provides the TokenReader method
// which takes control of the stream (preventing Serve from calling its
// handlers) and allows for full control over the incoming stream.
// To send XML on the output stream, Session has a TokenWriter method that
// returns a token encoder that holds a lock on the output stream until it is
// closed.
//
// Writing individual XML tokens can be tedious and error prone.
// The stanza package contains functions and structs that aid in the
// construction of message, presence and info/query (IQ) elements which have
// special semantics in XMPP and are known as "stanzas".
// There are 16 methods on Session used for transmitting stanzas and other
// events over the output stream.
// Their names are matched by the regular expression:
//
//     (Send|Encode)(Message|Presence|IQ)?(Element)?
//
// There are also four methods specifically for sending IQs and handling their
// responses.
// Their names are matched by:
//
//     (Unmarshal|Iter)IQ(Element)?
//
// If "Send" is present it means that the method copies one XML token stream
// into the output stream, while "Encode" indicates that it takes a value and
// marshals it into XML.
// If "IQ" is present it means that the stream or value contains an XMPP IQ and
// the method blocks waiting on a response.
// If "Element" is present it indicates that the stream or struct is a payload
// and not the full element to be transmitted and that it should be wrapped in
// the provided start element token or stanza.
//
//     // Send initial presence to let the server know we want to receive messages.
//     _, err = session.Send(context.TODO(), stanza.Presence{}.Wrap(nil))
//
// For SendIQ and related methods to correctly handle IQ responses, and to make
// the common case of polling for incoming XML on the input stream—and possibly
// writing to the output stream in response—easier, we need a long running
// goroutine.
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
package xmpp // import "mellium.im/xmpp"
