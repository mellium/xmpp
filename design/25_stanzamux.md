# Proposal: Implement stanza handlers

**Author(s):** Sam Whited  
**Last updated:** 2020-02-26  
**Discussion:** https://mellium.im/issue/25


## Abstract

An API is needed for multiplexing based on stanzas that allows for matching
based on type and quickly replying or re-marshaling the original stanzas.


## Background

In the previous [IQ mux proposal] a new mechanism for routing IQ stanzas based
on their type and payload was introduced.
In practice, registering the IQ muxer ended up being cumbersome, and the
previous proposal did not solve the problem of routing message or presence
stanzas (this was deliberately left for a future proposal).
To solve both of these problems at once the current [multiplexer][ServeMux] can
be adapted using what we learned from the IQ mux experiment such that it can
route message and presence stanzas based on their type, and IQs based on their
type and payload.


## Requirements

 - Ability to multiplex stanzas by stanza type and payload name
 - IQ, message, presence, and general top-level stream elements must have their
   own distinct handler types
 - The handlers must be extensible to add functionality such as replying to IQs
   or other thought of features in the future without making breaking API
   changes to the handler itself
 - The entire stanza and all children must be able to be decoded in one pass
   with [`Decoder.Decode`] or [`Decoder.DecodeElement`]


## Proposal

In addition to the existing [`Handler`] and [`IQHandler`] types, the following
types would be added for messages and presence:

    type MessageHandler interface {
            HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error
    }
        MessageHandler responds to message stanzas.

    type PresenceHandler interface {
            HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error
    }
        PresenceHandler responds to presence stanzas.


Adapters for functions with the provided signature would also be made available,
similar to [`HandlerFunc`].

Decoding the entire payload using an [`xml.Decoder`] requires the initial start
token which has already been consumed when creating the [`stanza.Message`],
[`stanza.IQ`], or [`stanza.Presence`].
We can't pass the start element instead of the decoded stanza type because this
limits future extensibility (we can add methods to the stanza type related to
encoding and decoding, replying, etc.), and it is undesirable to pass both
because then we are passing duplicate information to the handler.
Instead, we can prepend the start element to the buffered token stream so that
every handler has access to the entire stanza.

The existing IQ mux would be removed (this is a backwards incompatible change,
however, we are pre-1.0 and the IQ mux was never in a release) and its methods
and functionality would be added to [`ServeMux`][ServeMux].

The options related to registering stanzas would then be modified to take the
new patterns as follows:

    func IQ(typ stanza.IQType, payload xml.Name, h IQHandler) Option
        IQ returns an option that matches IQ stanzas by type and payload name.

    func Message(typ stanza.MessageType, payload xml.Name, h MessageHandler) Option
        Message returns an option that matches message stanzas by type and
        payload name.

    func Presence(typ stanza.PresenceType, payload xml.Name, h PresenceHandler) Option
        Presence returns an option that matches presence stanzas by type and
        payload name.

Functional versions of these options (taking a `HandlerFunc`) would also be
added.

Registering a handler that matches a stanza using the [`Handle`] option will
cause a panic, but this behavior is subject to change in the future.


[IQ mux proposal]: https://mellium.im/design/18_iqmux
[`Decoder.Decode`]: https://golang.org/pkg/encoding/xml/#Decoder.Decode
[`Decoder.DecodeElement`]: https://golang.org/pkg/encoding/xml/#Decoder.DecodeElement
[`Handler`]: https://pkg.go.dev/mellium.im/xmpp#Handler
[`IQHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#IQHandler
[`HandlerFunc`]: https://pkg.go.dev/mellium.im/xmpp#HandlerFunc
[`stanza.Message`]: https://pkg.go.dev/mellium.im/xmpp/stanza#Message
[`stanza.IQ`]: https://pkg.go.dev/mellium.im/xmpp/stanza#IQ
[`stanza.Presence`]: https://pkg.go.dev/mellium.im/xmpp/stanza#Presence
[`xml.Decoder`]: https://golang.org/pkg/encoding/xml/#Decoder
[ServeMux]: https://pkg.go.dev/mellium.im/xmpp/mux#ServeMux
[`Handle`]: https://pkg.go.dev/mellium.im/xmpp/mux#Handle
