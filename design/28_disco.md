# Proposal: Implement service discovery

**Author(s):** Sam Whited  
**Last updated:** 2020-11-15  
**Discussion:** https://mellium.im/issue/28


## Abstract

An API should be designed to handle service discovery that integrates with the
[`mux`] package.


## Background

Even the simplest client or server needs to discover information about other
entities on the network.
Because of this adding an API for service discovery improves our user experience
for almost all XMPP related projects including clients, servers, bots, and
embedded devices.
In the XMPP world service discovery is handled by [XEP-0030: Service Discovery]
and augmented by [XEP-0115: Entity Capabilities], but only the first will be
targeted by this design.


## Requirements

 - Ability to query for disco info and items
 - An API to walk the info and item tree
 - Ability to register features and items to a registry that can respond to
   disco info and disco items requests
 - Ability to register handlers on a multiplexer and have a registry of features
   automatically created from the set of handlers
 - Implementation must not preclude the addition of XEP-0115: Entity
   Capabilities support at a later date


## Proposal

A new package, `disco` will be created to handle registering features and
responding to disco info and items requests.
This package will comprise two new types, one new method, and 4 new functions
that will need to remain backwards compatible once we reach 1.0.
Predefined categories from the [disco categories registry][registry] may also be
generated but will not follow the normal compatibility promise since they will
be kept up to date with the registry.

```go
// A Registry is used to register features supported by a server.
type Registry struct {}

// NewRegistry creates a new feature registry with the provided identities and
// features.
// If multiple identities are specified, the name of the registry will be used
// for all of them.
func NewRegistry(...Option) *Registry {}

// HandleIQ handles disco info and item requests.
func (*Registry) HandleIQ(stanza.IQ, xmlstream.TokenReadEncoder, *xml.StartElement) error {}

// An Option is used to configure new registries.
type Option func(*Registry)

// Identity adds an identity to the registry.
func Identity(category, typ, name, lang string) Option {}

// Feature adds a feature to the registry.
func Feature(name string) Option {}

// Merge adds all features from the provided registry into the registry that the
// option is applied to.
func Merge(*Registry) Option {}
```

To make constructing a service discovery registry easier, the [`mux`] package
will also be modified with a new interface that can be implemented by handlers
to automatically register themselves with a built in disco registry.
This will require one new type, one new function, and one new method that will
need to remain backwards compatible once we reach version 1.0.

```go
// DiscoHandler is the type implemented by handlers that can be registered in a
// service discovery registry.
type DiscoHandler interface {
	Disco(*disco.Registry)
}

// Disco adds the provided options to the built in service discovery registry
// and then responds to disco info requests using the registry.
func Disco(opts ...disco.Option) Option {}

// DiscoRegistry returns a service discovery registry containing every feature
// and item from handlers registered on the mux that also supports the
// DiscoHandler interface.
func (*ServeMux) DiscoRegistry() *disco.Registry {}
```


## Open Questions

- How do we handle disco items requests?


[`mux`]: https://pkg.go.dev/mellium.im/xmpp/mux
[XEP-0030: Service Discovery]: https://xmpp.org/extensions/xep-0030.html
[XEP-0115: Entity Capabilities]: https://xmpp.org/extensions/xep-0115.html
[registry]: https://xmpp.org/registrar/disco-categories.html
