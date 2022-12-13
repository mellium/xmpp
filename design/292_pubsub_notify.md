# Pub-Sub Subscriptions

**Author(s):** Sam Whited <sam@samwhited.com>  
**Last updated:** 2022-12-13  
**Discussion:** https://mellium.im/issue/292

## Abstract

The [`pubsub` package] does not currently support the "sub" part of "pub-sub".
This proposal will detail a [mid-level API] to subscribe to nodes and act on
events that can be used by other packages that use pubsub.


[`pubsub` package]: https://pkg.go.dev/mellium.im/xmpp/pubsub
[mid-level API]: https://mellium.im/docs/ARCHITECTURE


## Terminology

This section includes any terminology that needs to be defined to understand the
rest of the document. This SHOULD include the latest text from [BCP 14]:

    The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL
    NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED",
    "MAY", and "OPTIONAL" in this document are to be interpreted as
    described in BCP 14 [RFC2119] [RFC8174] when, and only when, they
    appear in all capitals, as shown here.

[BCP 14]: https://tools.ietf.org/html/bcp14


## Background

Supporting subscribing to nodes, removing subscriptions, and receiving (and
handling) events and updates is a critical pre-requisite to many features that
use pubsub such as [OMEMO] ([#291]).
The existing pubsub functionality allows users to publish and fetch data,
effectively replacing the functionality of the legacy [`privatexml`] package,
but does not implement the extra features that make pubsub so valuable and which
are required by many of the specifications depending on it.

[OMEMO]: https://xmpp.org/extensions/xep-0384.html
[#291]: https://mellium.im/issue/291
[`privatexml`]: https://pkg.go.dev/mellium.im/legacy/privatexml


## Requirements

- The API MUST be able to subscribe to a pubsub node
- The API MUST have the ability to multiplex incoming pubsub events to
  different handlers based on payload namespace
- The API MUST NOT require dependent packages to process XML directly except to
  unmarshal individual payload items
- The API MUST allow unsubscribing from an existing subscription


## Proposal

```
// Subscriptions is a handler that, when registered against a mux with Handle,
// allows the user to subscribe and receive updates to pubsub nodes.
type Subscriptions struct { … }

// ForFeatures implements info.FeatureIter.
func (s *Subscriptions) ForFeatures(node string, f func(info.Feature) error) error { … }

// HandleMessage satisfies mux.MessageHandler.
func (s *Subscriptions) HandleMessage(p stanza.Message, r xmlstream.TokenReadEncoder) error { … }
```

The functionality of subscribing to pubsub events will be handled by a new type,
`Subscriptions`.
Like most extensions, it will be a handler that can be registered against a
multiplexer.
In this case it will handle incoming messages containing `<event/>` payloads
in the pubsub namespace.
The features iterator will advertise the `+notify` variant of any features
registered against the type to ensure that they are advertised by `disco#info`
and entity capabilities requests.
The message handler will also handle pubsub events such as subscription
confirmations and subscription removals (which may be in response to a request
from the user, or a directive from the server).

```
// Handle returns an option that registers the handler for use with a
// multiplexer.
func Handle(s *Subscriptions) mux.Option { … }
```

Like most extensions, pubsub will be handled by registering itself against a
multiplexer using a `Handle` function.

```
// SubType represents the state of a particular subscription.
type SubType uint8

func (SubType) String() string { … }

// UnmarshalXMLAttr satisfies xml.UnmarshalerAttr.
func (*SubType) UnmarshalXMLAttr(attr xml.Attr) error { … }

// MarshalXMLAttr satisfies xml.MarshalerAttr.
func (s *SubType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) { … }

// A list of possible subscription types.
const (
	SubNone         SubType = iota // none
	SubSubscribed                  // subscribed
	SubUnconfigured                // unconfigured
	SubPending                     // pending
)

// Subscription is a description of a particular subscription for which we will
// receive events.
type Subscription struct {
	ID             string
	Node           string
	Addr           jid.JID
	Subscription   SubType
	Configurable   bool
	ConfigRequired bool
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (Subscription) TokenReader() xml.TokenReader { … }

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (Subscription) WriteXML(w xmlstream.TokenWriter) (n int, err error) { … }

// MarshalXML implements xml.Marshaler.
func (Subscription) MarshalXML(e *xml.Encoder, _ xml.StartElement) error { … }

// UnmarshalXML implements xml.Unmarshaler.
func (*Subscription) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error { … }
```

The `Subscription` type is returned as the result of subscribing to a pubsub
node and provides the user with information about a specific subscription.
Its `String` method will be generated by the `stringer` tool and `go generate`.

```
// Handler responds to pubsub events.
type Handler interface {
    HandleEvent(stanza.Message, pubsub.Iter) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// event handlers.
// If f is a function with the appropriate signature, EventHandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(stanza.Message, pubsub.Iter) error

// HandleEvent calls f(msg, iter).
func (f HandlerFunc) HandleEvent(msg stanza.Message, iter pubsub.Iter) error { … }
```

The handler type gets registered against a pubsub service.
It is implemented by types in external packages to allow them to be registered
against a pubsub service and respond to events.
The message on the event handler is passed in so that a single handler can be
registered against multiple PEP or pubsub services and distinguish between
different senders.

```
// Subscribe creates a new subscription to a namespace.
//
// The namespace should not include the +notify suffix.
// If a subscription already exists for the namespace, Subscribe is a noop.
func (*Subscriptions) Subscribe(ctx context.Context, to jid.JID, session *xmpp.Session, ns string, h Handler) (Subscription, error) { … }

// SubscribeFunc is like Subscribe except it takes a HandlerFunc.
func (*Subscriptions) SubscribeFunc(ctx context.Context, to jid.JID, session *xmpp.Session, ns string, h Handler) (Subscription, error) { … }

// Unsubscribe removes a subscription and stops handling events.
func (*Subscriptions) Unsubscribe(ctx context.Context, ns string) error { … }
```

The actual act of subscribing will be performed using a method that adds a
handler and begins forwarding events to it.
Unsubscribing will send the unsubscribe stanza regardless of whether a
subscription is registered (in case of server policy automatically subscribing a
user to some events), but may return an error if the server sends one back.
Similarly, calling Subscribe will always send the subscribe stanza, replacing
the handler if it exists locally and no error is returned from the subscription
response.

```
// Handler returns the handler to use for an item list with the given namespace.
// If no exact match is found, a default noop handler is returned (h is always
// non-nil) and ok will be false.
func (*Subscriptions) Handler(ns string) (h Handler, ok bool)
```

As a convenience (and to more closely match the API provided by the
multiplexer) the `Handler` function returns whether a handler is registered for
a given subscription.

Errors returned from pubsub methods will return a special error type that
contains information about the underlying XML errors and embeds a stanza.Error.

```
// Error represents an error returned by a pubsub service.
type Error struct {
	stanza.Error
	Condition Condition

	// The specific feature that caused the error.
	// Only set if the Condition is Unsupported.
	Feature string

	// The new location of a pubsub node that has been moved.
	// Only set if the Condition is Gone.
	NewURI string

	// The required configuration for the node.
	// Only set if the Condition is ConfigurationRequired.
	Configuration *form.Data
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (Error) TokenReader() xml.TokenReader { … }

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (Error) WriteXML(xmlstream.TokenWriter) (int, error) { … }

// MarshalXML implements xml.Marshaler.
func (Error) MarshalXML(*xml.Encoder, xml.StartElement) error { … }

// UnmarshalXML implements xml.Unmarshaler.
func (*Error) UnmarshalXML(*xml.Decoder, xml.StartElement) error { … }

// Error implements the error interface.
func (Error) Error() string { … }

// Condition is the underlying cause of a pubsub error.
type Condition uint32

const (
	// Subscribe errors
	InvalidAddr           Condition = iota // invalid-jid
	PresenceRequired                       // presence-subscription-required
	NotInRoster                            // not-in-roster-group
	ClosedNode                             // closed-node
	PaymentRequired                        // payment-required
	Pending                                // pending-subscription
	ResourceUsage                          // too-many-subscriptions
	Unsupported                            // unsupported
	Gone                                   // gone
	NotFound                               // item-not-found
	ConfigurationRequired                  // configuration-required

	// Unsubscribe errors
	IDRequired    // subid-required
	IDInvalid     // invalid-subid
	NotSubscribed // not-subscribed
)
```

Finally, following our standard procedures, the namespaces will be exported:

```
const (
	NSEvent  = `http://jabber.org/protocol/pubsub#event`
	NSErrors = `http://jabber.org/protocol/pubsub#errors`
    NSOptions = `http://jabber.org/protocol/pubsub#subscription-options`
)
```

Implementing this proposal would add seven types, 19 methods, 21 constants,
and one function that would need to remain backwards compatible once we reach
1.0.

## Open Issues

- How does notification filtering fit into this API?
