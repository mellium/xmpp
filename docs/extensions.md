# Implementing XMPP Extensions

The [`mellium.im/xmpp`] module contains a number of packages that implement
[XMPP Extension Protocols][XEP] (XEPs).
When contributing new packages, or publishing your own proprietary extensions
separately, it's important to make sure that your package is consistent with
existing code.
The rules in this post were designed to facilitate consistent extension packages
and should always be followed when contributing new packages upstream.

## Group XEPs into packages by functionality

Normally this means that each XEP is given its own package.
However, for smaller or tightly coupled XEPs this may mean sharing a package.
For instance, [XEP-0082: XMPP Date and Time Profiles][XEP-0082] and [XEP-0202:
Entity Time][XEP-0202] are both related to the concept of transmitting time over
an XMPP connection, and XEP-0082 is effectively just a list of constants.
Because of this, they both share a single package,
[`mellium.im/xmpp/xtime`][`mellium.im/xmpp/xtime`] (note that `xtime` would not
normally be considered a good package name, `time` would be better but this
conflicts with a package in the standard library).
Similarly, the various XEPs that make up the Jingle signaling protocol would
likely be grouped together.
Meanwhile [XEP-0234: Jingle File Transfer][XEP-0234] and [XEP-0047: In-Band
Bytestreams][XEP-0047] are both about file transfer, but they likely would *not*
share a single package as they wouldn't share much code whereas XEP-0234 likely
*would* share code with other Jingle related XEPs.

When in doubt, [ask][support].
There are no hard and fast rules to grouping XEPs into packages.


## Implement data transmissions with simple functions

Users of your package shouldn't have to think about XML, stanzas, or XMPP.
Instead, we should provide our users with simple functions that can be used to
perform basic operations over the network, and which result in common types
where possible.
For example, the [`xtime.Get`] function sends an IQ requesting the time to the
given address over the given session and returns an ordinary [`time.Time`]:

    func Get(ctx context.Context, s *xmpp.Session, to jid.JID) (time.Time, error)
        Get sends a request to the provided JID asking for its time.

Under the hood this might be sending an IQ and blocking while it waits for a
response, or returning a payload that had been previously decoded by a message
handler.
The user doesn't need to know the details, all that they care about is asking
for a time and getting one in the format they're already used to dealing with.


## Use the `mux` package to handle incoming data

The [`mellium.im/xmpp/mux`] package provides an [`xmpp.Handler`] that can
multiplex various top level elements to custom handlers based on their name or
stanza type and the name of a payload.
While other multiplexers may exist in the future, they should all use the
standard handler types:

- [`xmpp.Handler`] for non-stanza top level elements or custom muxers
- [`mux.IQHandler`]
- [`mux.MessageHandler`]
- [`mux.PresenceHandler`]

Because of this, if your package handles one or more types of payload it should
contain a type, generally called `Handler` (eg. [`xtime.Handler`] or
[`ping.Handler`]) that implements one or more of the above handler interfaces
and can be registered with a [`mux.ServeMux`] to respond to incoming payloads.

Similarly, your package should contain a [`mux.Option`] that registers the
handler.
Generally this option will be called `Handle`, but if multiple handlers exist
that need to be registered separately more descriptive names may be necessary
such as `example.HandleData` and `example.HandleOpen`.
This can either create a new handler if the handler does not require any
configuration as the [`ping.Handle`] option does, or take a specific handler to
register if further configuration may be required as the [`xtime.Handle`] option
does.
Options configure the `mux.ServeMux` using fields that are not exported, so they
must be constructed in terms of other, existing, options.
For example, the handler form `xtime` only handles a specific type of IQ and is
defined as:

    // Handle returns an option that registers a Handler for entity time requests.
    func Handle(h Handler) mux.Option {
    	return mux.IQ(stanza.GetIQ, xml.Name{Local: "time", Space: NS}, h)
    }

A handler that needs to handle multiple types can use a raw `mux.Option` func to
group multiple options into one:

    // Handle is an option that registers an example handler that implements
    // both IQHandler and MessageHandler.
    func Handle(h Handler) mux.Option {
    	return func(m *ServeMux) {
    		mux.IQ(stanza.GetIQ, xml.Name{Local: "example", Space: NS}, h)(m)
    		mux.Message(stanza.NormalMessage, xml.Name{Local: "example", Space: NS}, h)(m)
    	}
    }


## Export structs for marshaling and unmarshaling

Normally the user will want to use the functions and handlers defined in your
package to transmit and receive data, but sometimes they may need to reimplement
your extension to add their own proprietary extensions, or to instrument the
code for a required logging or metrics library.
So that users can reimplement as little as possible, and for the sake of
consistency between implementations, we export structs that can be used to
marshal or unmarshal stanzas and payloads.

Many XEPs will likely have a single payload that needs to be exported.
In this case, if the payload does not need any custom logic to marshal or
unmarshal, a struct representing the full IQ (or other stanza type) should be
exported.
For example, the [`mellium.im/xmpp/ping`] package contains the definition for a
ping IQ:

    // IQ is encoded as a ping request.
    type IQ struct {
    	stanza.IQ

    	Ping struct{} `xml:"urn:xmpp:ping ping"`
    }

Exported stanza types should be named after the type of stanza they implement.
The name of the previous examples type when fully qualified is therefore
[`ping.IQ`].
If multiple stanza types need to be exported from the same package, give them
all descriptive names, for example: `example.RequestMessage` and
`example.ResponseMessage`.

If one or more payloads exist that need custom logic to marshal or unmarshal,
export the individual payload type with implementations of [`xml.Marshaler`]
and [`xml.Unmarshaler`] containing the custom logic.

In either case, the exported stanza or payload types should implement the
following interfaces:

- [`xmlstream.Marshaler`]
- [`xmlstream.WriterTo`], implemented in terms of `xmlstream.Marshaler`
- [`xml.Marshaler`], implemented in terms of `xmlstream.WriterTo` if custom
  marshaling logic is required
- [`xml.Unmarshaler`] if custom unmarshaling logic is required

For an example, see the source for [`xtime.Time`].


## Lazily decode large, repeating payloads

When unmarshaling a payload of unknown length that contains many similar
children, an iterator should be written instead of unmarshaling them all into a
slice.
This gives the user the option of keeping memory usage low by reusing values,
reducing copying when the length is unknown and a slice would have to be
extended, and keeping CPU usage low by only partially decoding the child
elements if we can short circuit after finding a particular child.

The iterator type should be called `Iter` and can be used to lazily decode
children into a type the user can deal with.
The `Iter` type should have an API similar to the following example, taken from
the [`roster`] package:

```
// Iter is an iterator over roster items.
type Iter struct{}

// Item returns the last roster item parsed by the iterator.
(i *Iter) Item() Item

// Next reports whether there are more items to decode.
(i *Iter) Next() bool

// Err returns the last error encountered by the iterator (if any).
(i *Iter) Err() error

// Close indicates that we are finished with the given iterator and processing
// the stream may continue.
// Calling it multiple times has no effect.
(i *Iter) Close() error
```

The `Item` method and its return type are named after the thing being
decoded, in this case a roster item, and may be different for every `Iter`.
From the developers perspective, the `Iter` type can now decode elements from
the stream with a simple for loop:

```
iter := roster.Fetch(context.TODO(), session)
defer iter.Close()

for iter.Next() {
	item := iter.Item()
	// Do something with the roster item
}
if iter.Err() != nil {
	// Handle errors
}
```

Because iterators are common and all largely share the same logic to decode
child elements and return them, the [`internal/iter`] package was written to
make much of the logic reusable.
Because this package is internal you can't use it for your custom extensions
yet, but its types will be moved to an external package once we are sure that
the API is stable.
Instead of operating directly on decoded child elements, the `iter` package
operates on the token stream and returns access to each child element it finds,
letting your `Iter` type do the final decoding into a concrete type of your
choosing.
In the case of the `roster` package, this type is a `roster.Item`.

Most iterators will need to maintain some internal state.
This normally comprises any errors that were generated, a value representing the
last child element that was decoded, and an underlying `[`iter.Iter`].

On your `Iter` type, most methods need to simply call the similarly named method
on the underlying `iter.Iter`.
The exceptions are the `Err` and `Next` methods.
The `Err` method should return any errors generated by your decoding first and
if no such errors exist return `iter.Err()`.
The `Next` method is a bit more complicated.
It should first check that no previous decoding errors exist and that
`iter.Next()` is true.
If this is the case it should decode the tokens provided by `iter.Current()` and
save the result or any errors generated.

For example, the `roster` package defines `Next` like so:

```
// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	start, r := i.iter.Current()
	d := xml.NewTokenDecoder(r)
	item := Item{}
	i.err = d.DecodeElement(&item, start)
	if i.err != nil {
		return false
	}
	i.current = item
	return true
}
```


[`internal/iter`]: https://pkg.go.dev/mellium.im/xmpp/internal/iter
[`iter.Iter`]: https://pkg.go.dev/mellium.im/xmpp/internal/iter#Iter
[`mellium.im/xmpp`]: https://pkg.go.dev/mellium.im/xmpp/mux
[`mellium.im/xmpp/mux`]: https://pkg.go.dev/mellium.im/xmpp/mux
[`mellium.im/xmpp/ping`]: https://pkg.go.dev/mellium.im/xmpp/ping
[`mellium.im/xmpp/xtime`]: https://pkg.go.dev/mellium.im/xmpp/xtime
[`mux.IQHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#IQHandler
[`mux.MessageHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#MessageHandler
[`mux.Option`]: https://pkg.go.dev/mellium.im/xmpp/mux#Option
[`mux.PresenceHandler`]: https://pkg.go.dev/mellium.im/xmpp/mux#PresenceHandler
[`mux.ServeMux`]: https://pkg.go.dev/mellium.im/xmpp/mux#ServeMux
[`ping.Handle`]: https://pkg.go.dev/mellium.im/xmpp/ping#Handle
[`ping.Handler`]: https://pkg.go.dev/mellium.im/xmpp/ping#Handler
[`ping.IQ`]: https://pkg.go.dev/mellium.im/xmpp/ping#IQ
[`roster`]: https://pkg.go.dev/mellium.im/xmpp/roster
[`roster.Item`]: https://pkg.go.dev/mellium.im/xmpp/roster#Item
[support]: https://mellium.im/docs/SUPPORT
[`time.Time`]: https://golang.org/pkg/time/#Time
[XEP-0047]: https://xmpp.org/extensions/xep-0047.html
[XEP-0082]: https://xmpp.org/extensions/xep-0082.html
[XEP-0202]: https://xmpp.org/extensions/xep-0202.html
[XEP-0234]: https://xmpp.org/extensions/xep-0234.html
[XEP]: https://xmpp.org/extensions/
[`xml.Marshaler`]: https://golang.org/pkg/encoding/xml/#Marshaler
[`xmlstream.Marshaler`]: https://pkg.go.dev/mellium.im/xmlstream#Marshaler
[`xmlstream.WriterTo`]: https://pkg.go.dev/mellium.im/xmlstream#WriterTo
[`xml.Unmarshaler`]: https://golang.org/pkg/encoding/xml/#Unmarshaler
[`xmpp.Handler`]: https://pkg.go.dev/mellium.im/xmpp#Handler
[`xtime.Get`]: https://pkg.go.dev/mellium.im/xmpp/xtime#Get
[`xtime.Handle`]: https://pkg.go.dev/mellium.im/xmpp/xtime#Handle
[`xtime.Handler`]: https://pkg.go.dev/mellium.im/xmpp/xtime#Handler
[`xtime.Time`]: https://pkg.go.dev/mellium.im/xmpp/xtime#Time
