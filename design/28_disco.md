# Proposal: Implement service discovery

**Author(s):** Sam Whited  
**Last updated:** 2021-08-10  
**Discussion:** https://mellium.im/issue/28


## Abstract

An API should be designed to handle responding to service discovery requests
that integrates with the [`mux`] package.


## Background

Even the simplest client or server needs to discover information about other
entities on the network.
Because of this adding an API for responding to service discovery improves our
user experience for almost all XMPP related projects including clients, servers,
bots, and embedded devices.
In the XMPP world service discovery is handled by [XEP-0030: Service Discovery]
and augmented by [XEP-0115: Entity Capabilities], but only the first will be
targeted by this design.


## Requirements

 - Ability to register handlers on a multiplexer and have a registry of features
   automatically created from the set of handlers
 - Implementation must not preclude the addition of items or XEP-0115 support at
   a later date


## Proposal

A new package, `disco/info` will be created and `disco.Feature` will be moved to
`info.Feature` to avoid import loops between the `mux` and `disco` packages.
This is a breaking API change, but is acceptable as we are currently before
version 1.0.

The info package will contain interfaces that can be implemented by handlers
providing features:

```go
// FeatureIter is the interface implemented by types that implement disco
// features.
type FeatureIter interface{
  ForFeatures(node string, f func(Feature) error) error
}
```

To make responding to service discovery requests easier, the [`mux`] package
will also be modified to implement this new interface.

```go
// ForFeatures implements info.FeatureIter for the mux by iterating over all
// child features.
func (m *ServeMux) ForFeatures(node string, f func(info.Feature) error) error
```

Finally, the `disco` package will be given a new `muc.Option` that responds to
disco requests by calling `m.ForFeatures` method:

```go
// Handle returns an option that configures a multiplexer to handle service
// discovery requests by iterating over its own handlers and checking if they
// implement info.FeatureIter.
func Handle() mux.Option
```

Overall this will result in 1 new type, one new method, and one new function
that will need to remain backwards compatible once we reach 1.0.


[`mux`]: https://pkg.go.dev/mellium.im/xmpp/mux
[XEP-0030: Service Discovery]: https://xmpp.org/extensions/xep-0030.html
[XEP-0115: Entity Capabilities]: https://xmpp.org/extensions/xep-0115.html
