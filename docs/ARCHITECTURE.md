# Architecture

This document provides a high-level architectural overview of the
[`mellium.im/xmpp`] module.
Its target audience is contributors looking to familiarize themselves with the
codebase.
The Mellium project is designed to be very modular.
It is itself a complete XMPP implementation with many features useful in instant
messaging, but it is also designed for users who have special requirements to be
able to reuse only small pieces in their own libraries and services.

Broadly speaking, the module can be divided into three 'tiers' of packages.
The highest level packages implement [XMPP Extension Protocols][XEP] (XEPs) or
individual features.
These packages import the mid-level [`xmpp`] package to perform actions on an
XMPP session such as handling a request for history, or sending a ping.
High level packages and the mid-level `xmpp` package import the lowest level of
packages.
These are the basic building blocks of XMPP such as the primitive stanza types
defined in [`stanza`], dialing a socket with [`dial`], or the [`jid`] package
for handling XMPP addresses.
Packages may only import packages at the same or a lower level to avoid circular
imports.

![Dependency graph](https://mellium.im/localdeps.svg)

These boundaries aren't strictly defined, but it can be helpful to think about
them when creating new packages or deciding where to implement some
functionality.

## Low-level Packages

### Connections and Service Discovery

Dialing TCP connections is implemented in its own package, [`dial`].
Discovering locations to dial is handled by [`internal/discover`] so that it can
be used by `dial` for finding TCP endpoints but also by the [`websocket`]
package for finding WebSocket endpoints.


### XMPP Addresses

The [`jid`] package contains functionality for handling XMPP addresses,
historically known as "Jabber Identifiers" (JIDs).
It is used by almost every package in the module and will likely be imported by
almost every user of the module.

### Stanzas, Errors, and Streams

Multiple APIs for creating stanzas and stanza-level errors are present in the
[`stanza`] package.
APIs for creating and parsing stream headers and stream-level errors are present
in the [`stream`] package.
If you are handling anything related to payloads sent over the wire (but not
related to a specific XEP), chances are it lives in one of these two packages or
in the related [`internal/stream`] package which contains other stream related
helper functions that aren't generally useful or flexible enough to be part of
the public API.

## Mid-level Packages

### Sessions and Feature Negotiation

The main [`xmpp`] package contains just enough functionality to get a connection
up and running.
Within this package the main files are:

- `session.go`  which includes the session creation and negotiation logic,
- `negotiator.go` which implements the default handshake used when creating
  sessions and the modified WebSocket handshake, and
- `features.go` which contains types for creating stream features and the logic
  used by the default negotiator pick and negotiate individual features.

Individual stream features are in separate files that match their name such as
`bind.go` or `starttls.go`.

### Multiplexing

The [`mux`] package is also arguably a mid-level package.
It is imported by most high-level feature packages and provides an
[`xmpp.Handler`] that can be used to route events to other handlers based on the
type of the element and/or its payload.
If you are asked to make modifications to routing logic, this is the place to
look.


## High-level Packages

### Other XEPs and Features

XEPs are normally implemented in packages named after their functionality.
This may be the XEPs name or short name if it is large enough to warrant its own
package.
For example, [XEP-0313: Message Archive Management][XEP-0313] might be
implemented in the `mam` package.
Smaller XEPs or large features that are spread across many XEPs may be
implemented together, for example, [XEP-0082: XMPP Date and Time
Profiles][XEP-0082] and [XEP-0202: Entity Time][XEP-0202] are both implemented
in the [`xtime`] package.

For more information on naming and writing feature packages see "[Implementing
XMPP Extensions]".
If you are looking for a specific XEP and the package that implements it see:
[mellium.im/docs/xeps].

[`dial`]: https://pkg.go.dev/mellium.im/xmpp/dial
[Implementing XMPP Extensions]: https://mellium.im/docs/extensions
[`internal/discover`]: https://pkg.go.dev/mellium.im/xmpp/internal/discover
[`internal/stream`]: https://pkg.go.dev/mellium.im/xmpp/internal/stream
[`jid`]: https://pkg.go.dev/mellium.im/xmpp/jid
[mellium.im/docs/xeps]: https://mellium.im/docs/xeps
[`mellium.im/xmpp`]: https://mellium.im/xmpp
[`mux`]: https://pkg.go.dev/mellium.im/xmpp/mux
[`stanza`]: https://pkg.go.dev/mellium.im/xmpp/stanza
[`websocket`]: https://pkg.go.dev/mellium.im/xmpp/websocket
[XEP-0082]: https://xmpp.org/extensions/xep-0082.html
[XEP-0202]: https://xmpp.org/extensions/xep-0202.html
[XEP-0313]: https://xmpp.org/extensions/xep-0313.html
[XEP]: https://xmpp.org/extensions/
[`xmpp.Handler`]: https://pkg.go.dev/mellium.im/xmpp#Handler
[`xmpp`]: https://pkg.go.dev/mellium.im/xmpp
[`xtime`]: https://pkg.go.dev/mellium.im/xmpp/xtime
