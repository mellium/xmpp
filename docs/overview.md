# Mellium Overview

Go is a great language if you need to get a project off the ground quickly. It
has its confusing aspects, and its type system allows for lots of abuse thanks
to optional dynamic typing, but overall it's easy to read and easy to quickly
build projects that require clear code over absolute type safety.
Similarly, <abbr title="Extensible Messaging and Presence
Protocol">XMPP</abbr> (1) has its warts, but overall is the best choice to get a
chat product off the ground quickly if you want a system that's well understood
and has a robust ecosystem and sustainable standards body.

Since Go shines at handling I/O bound services (like asynchronous network
protocols used for instant messaging), an XMPP library in Go seems like a great
fit.
There are a handful of libraries to handle XMPP already in existence, but most
of them are small high-level libraries designed only to work with the legacy
version of XMPP that was supported by Google Talk, or don't follow go idioms and
best practices.
When I started looking into a Go XMPP implementation around 5 years ago, there
wasn't a low-level library meant to act as a building block from which higher
level systems could be created, and that's what I wanted: the equivalent of the
standard libraries [`net/http`] but for XMPP.
This is why I created [`mellium.im/xmpp`].
This post will be about some of the design decisions I made while building the
library, and about some of the trade offs made along the way.

(1): XMPP is sometimes referred to as "Jabber" for historical reasons. I prefer
      to refer to the federated network as Jabber and the protocol as XMPP.
      From this point on, Jabber is to Email what XMPP is to SMTP.


## Stream Features

Let's start by talking about feature negotiation.
An XMPP session can broadly be divided up into two parts: the synchronous
initial handshake, and the actual asynchronous session.
Within this initial handshake, a series of common features are negotiated in a
certain order.
For example, if TLS isn't already in use, opportunistic TLS (StartTLS) might be
negotiated, followed by authentication.
This ends up being a loop where the server sends any features it wants to
advertise at the current moment (eg. just TLS) then the client chooses one to
negotiate and proceeds with that features specific negotiation steps.
Then the server sends another list (possibly with new features, eg. now that we
have TLS negotiated the server might advertise that authentication is now ready
to proceed) and the client selects one and moves forward.
This loop is easy enough to write in Go, but representing the features
themselves was tricky.
Features need to be able to encode the name they go by when the server lists
them and any information that should be included in that listing, they need to
be able to parse that payload from the clients side, and the actual negotiation
from the server and clients side needs to happen.
To handle this a struct containing three functions for listing features, parsing
the features list, and negotiating the features was created.
This means less boilerplate and more type safety than using an interface to
represent a stream feature.
It also makes it less likely that a user of this API will get confused and write
stateful stream features, but if necessary the functions can still close over
external state or resources (but don't do this, you may think you need it, but
you're almost certainly wrong).

```
type StreamFeature struct {
	Name xml.Name

	Necessary SessionState
	Prohibited SessionState

	List func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error)
	Parse func(ctx context.Context, r xml.TokenReader, start *xml.StartElement) (req bool, data interface{}, err error)
	Negotiate func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, err error)
}
```

We also need to have a way to encode the order features should appear in (eg.
auth should not be attempted before TLS).
I decided that features would order themselves based on the state of the
session at the moment when feature negotiation happens.
The feature would say what properties of the session are or are not allowed, and
the thing doing negotiation can determine whether the session currently meets
those criteria.
Session state information only has 4 properties that are useful for session
negotiation:

- Is a security layer in place (eg. TLS),
- has authentication been performed,
- is feature negotiation complete, and
- was the session initiated by a remote entity?

These are part of the [`SessionState`] bits, so in the stream features we can
encode what bits are [necessary] and what bits are [prohibited] and the state
machine that handles session negotiation will be able to figure out when to
advertise or negotiate the feature using simple bit math.

```
const (
	// Secure indicates that the underlying connection has been secured. For
	// instance, after STARTTLS has been performed or if a pre-secured connection
	// is being used such as websockets over HTTPS.
	Secure SessionState = 1 << iota

	// Authn indicates that the session has been authenticated (probably with
	// SASL).
	Authn

	// Ready indicates that the session is fully negotiated and that XMPP stanzas
	// may be sent and received.
	Ready

	// Received indicates that the session was initiated by a foreign entity.
	Received

	…
)
```


[`SessionState`]: https://pkg.go.dev/mellium.im/xmpp#SessionState
[necessary]: https://pkg.go.dev/mellium.im/xmpp#StreamFeature.Necessary
[prohibited]: https://pkg.go.dev/mellium.im/xmpp#StreamFeature.Necessary


## Session Negotiation

Once we have a set of features that we can negotiate, we need to do the actual
session negotiation.
Normally, XMPP negotiates a session over TCP using the features loop that we
already described, however, sometimes an alternative mechanism might be required
for negotiation such as the websocket subprotocol defined in [RFC 7395] or the
legacy [XEP-0114: Jabber Component Protocol][XEP-0114].
Generalizing session negotiation meant allowing the user to provide a special
negotiator function and writing a default one for the basic XMPP stream
negotiation protocol.

```
type Negotiator func(ctx context.Context, session *Session, data interface{})
  (mask SessionState, rw io.ReadWriter, cache interface{}, err error)
```

Because the negotiator can't change the session state if it's written in another
package (since the session state bits aren't exported), it returns any changes
it wants to be made to the session such as the new session state mask, or any
changes to the underlying reader and writer (eg. if we negotiate StartTLS it
might return a new reader and writer that speak TLS).
The internal code that calls the negotiator function can then create a new
session with the requested changes.

The builtin negotiator can be created with [`NewNegotiator`] and supports
various options such as setting the stream language and copying the input and
output streams somewhere else (such as an XML console):

```
// StreamConfig contains options for configuring the default Negotiator. 
type StreamConfig struct {
	// The native language of the stream.
	Lang string

	// S2S causes the negotiator to negotiate a server-to-server (s2s) connection.
	S2S bool

	// A list of stream features to attempt to negotiate.
	Features []StreamFeature

	// If set a copy of any reads from the session will be written to TeeIn and
	// any writes to the session will be written to TeeOut (similar to the tee(1)
	// command).
	// This can be used to build an "XML console", but users should be careful
	// since this bypasses TLS and could expose passwords and other sensitve data.
	TeeIn, TeeOut io.Writer
}

// NewNegotiator creates a Negotiator that uses a collection of StreamFeatures
// to negotiate an XMPP client-to-server (c2s) or server-to-server (s2s)
// session.
// If StartTLS is one of the supported stream features, the Negotiator attempts
// to negotiate it whether the server advertises support or not.
func NewNegotiator(func(*Session, *StreamConfig) StreamConfig) Negotiator
```

It uses stream features as discussed in the previous
section, but custom negotiators could be written that use a different type for
stream features, making session negotiation and stream features entirely
modular.
You could replace them with your own implementations, and still use the `xmpp`
package to handle the lower level XMPP protocol.

An example of a custom stream negotiator can be found in the [`xmpp/component`]
package which negotiates a [XEP-0114: Jabber Component Protocol][XEP-0114]
connection.


[RFC 7395]: https://tools.ietf.org/html/rfc7395
[XEP-0114]: https://xmpp.org/extensions/xep-0114.html
[`xmpp/component`]: https://pkg.go.dev/mellium.im/xmpp/component
[`NewNegotiator`]: https://pkg.go.dev/mellium.im/xmpp#NewNegotiator


## Receiving Data

Once the session is negotiated, we need to be able to receive stanzas (the
primitive types of XMPP) and other top level XML elements over the session.
Because the main `xmpp` package is meant to be lower level than many other XMPP
libraries written for Go, it does not contain callbacks or any way to register
handlers for different types of top level XML element.

Instead, it contains a single [`Session.Serve`] method that decodes all incoming
XML tokens and delegates handling them to a single [`Handler`].

```
// A Handler triggers events or responds to incoming elements in an XML stream.
type Handler interface {
	HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error
}
```

The `Serve` method also handles stanza semantics such as always responding to
IQs as required to be compliant with the XMPP protocol.
Because the handler is provided with a stream to use when writing back to the
session, the underlying library can man-in-the-middle the token stream and check
if an IQ response was written and automatically send one if not, or add required
IDs to stanzas that are missing them.

This design also keeps the number of methods on the session relatively low and
keeps the entire library more modular because we can delegate multiplexing
elements to more specific handlers to other packages such as the builtin
[`xmpp/mux`] package.
If a user wants a more advanced multiplexer that buffers the stream and matches
stream elements based on [RELAX NG], [Clark notation], or [XPath], they could
write it themselves and it would have all the same powers and access as the
built in muxer!
The `mux` package also provides more specific handlers for the basic XMPP stanza
types: [`IQHandler`], [`MessageHandler`], and [`PresenceHandler`], which should
be reused by third party multiplexers.


[`Session.Serve`]: https://pkg.go.dev/mellium.im/xmpp#Session.Serve
[`Handler`]: https://pkg.go.dev/mellium.im/xmpp#Handler
[`xmpp/mux`]: https://pkg.go.dev/mellium.im/xmpp/mux
[RELAX NG]: https://relaxng.org/
[Clark notation]: https://web.archive.org/web/20200320131503/http://www.jclark.com/xml/xmlns.htm
[XPath]: https://www.w3.org/TR/xpath/all/
[`IQHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#IQHandler
[`MessageHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#MessageHandler
[`PresenceHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#PresenceHandler


## Sending Data

Naturally, receiving data isn't enough.
We also need to send it.
This happens by calling methods directly on the `Session`, the full list of
methods for sending data is:

- [`Encode`]`(v interface{}) error`
- [`EncodeElement`]`(v interface{}, start xml.StartElement) error`
- [`Send`]`(ctx context.Context, r xml.TokenReader) error`
- [`SendElement`]`(ctx context.Context, r xml.TokenReader, start xml.StartElement) error`
- [`SendIQ`]`(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error)`
- [`SendIQElement`]`(ctx context.Context, payload xml.TokenReader, iq stanza.IQ) (xmlstream.TokenReadCloser, error)`
- [`TokenWriter`]`() xmlstream.TokenWriteFlushCloser`

This collection of methods gives you a low level way to take out a lock on the
output stream and write tokens with `TokenWriter`, a high level way to send Go
types without worrying about the underlying XML (the `Encode` methods), and
methods for copying a token reader (provided by many types meant to be marshaled
to XML) with the `Send` methods.
The `SendIQ` methods differ a tiny bit from the other methods because all IQ
stanzas in XMPP receive a reply.
Having separate methods for `SendIQ` let you block a goroutine waiting for that
reply so that you can write asynchronous code in a synchronous style, which is
Go's super power.

One slightly confusing aspect of this is that the `Serve` goroutine mentioned in
the previous session must be running for the `SendIQ` methods to work.
This is because serve also handles receiving IQs and matching their IDs to the
list of sent IQs.
Though this is well documented, it is often confusing for new users of the
library.
In a future version of the library, the `SendIQ` methods may learn to return an
error if the server isn't running, but for now their behavior without the
`Serve` goroutine running is undefined (and will likely lead to them blocking
forever).


[`Encode`]: https://pkg.go.dev/mellium.im/xmpp#Session.Encode
[`EncodeElement`]: https://pkg.go.dev/mellium.im/xmpp#Session.EncodeElement
[`Send`]: https://pkg.go.dev/mellium.im/xmpp#Session.Send
[`SendElement`]: https://pkg.go.dev/mellium.im/xmpp#Session.SendElement
[`SendIQ`]: https://pkg.go.dev/mellium.im/xmpp#Session.SendIQ
[`SendIQElement`]: https://pkg.go.dev/mellium.im/xmpp#Session.SendIQElement
[`TokenWriter`]: https://pkg.go.dev/mellium.im/xmpp#Session.TokenWriter


## Conclusion

That's it! You should now be acquainted enough with the `xmpp` package to follow
the examples in the documentation and generally understand how the various more
advanced connection mechanisms we didn't discuss here work.
The module does so much more than just low-level XMPP connections though, and
more functionality can be found in the [subdirectories].
If you want to write your own extension (or learn more about why the existing
extensions were written the way that they have been), see the file
`extensions.md`.
Finally, be sure to [let me know] if you build anything interesting with
Mellium!

[subdirectories]: https://pkg.go.dev/mellium.im/xmpp?tab=subdirectories
[let me know]: xmpp:sam@samwhited.com?message

[`mellium.im/xmpp`]: https://pkg.go.dev/mellium.im/xmpp
[`net/http`]: https://golang.org/pkg/net/http/
