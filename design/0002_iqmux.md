# Proposal: Implement a method of muxing based on IQ payloads

**Author(s):** Sam Whited  
**Last updated:** 2020-02-13  
**Status:** thinking

## Abstract

An API is needed for multiplexing based on IQ payloads.


## Background

The current [multiplexer] only matches on top level elements. While the mux
could be nested to provide support for matching on element children, this would
mean that handlers written for children could accidentally be registered at the
top level muxer. Some other mechanism is needed to make it obvious that handlers
written to match against IQ child elements aren't meant to be registered against
the top level muxer.
To this end a new `IQMux` will be introduced that can be registered with the top
level multiplexer to handle all IQ elements and can take its own custom handlers
meant for matching IQ payloads.

[multiplexer]: https://pkg.go.dev/mellium.im/xmpp/mux#ServeMux


## Requirements

 - Ability to mux IQ stanzas based on XML name (and possibly other factors)
 - IQ handlers must be distinguishable from general XMPP handlers


## Proposal

The proposed API creates three new types and five new functions that would need
to remain backwards compatible after we reach 1.0.

    type IQHandler interface {
    	HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error
    }
        IQHandler responds to an IQ stanza.

    type IQHandlerFunc func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error
        The IQHandlerFunc type is an adapter to allow the use of ordinary functions
        as IQ handlers. If f is a function with the appropriate signature,
        IQHandlerFunc(f) is an IQHandler that calls f.

    func (f IQHandlerFunc) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error
        HandleIQ calls f(iq, t, start).

    type IQMux struct {
    	// Has unexported fields.
    }
        IQMux is an XMPP multiplexer meant for handling IQ payloads.

    func NewIQMux(opt ...IQOption) *IQMux
        NewIQMux allocates and returns a new IQMux.

    func (m *IQMux) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error
        HandleXMPP dispatches the IQ to the handler whose pattern most closely
        matches start.Name.

    func (m *IQMux) Handler(iqType stanza.IQType, name xml.Name) (h IQHandler, ok bool)
        Handler returns the handler to use for an IQ payload with the given name and
        type. If no handler exists, a default handler is returned (h is always
        non-nil).

    type IQOption func(m *IQMux)
        IQOption configures an IQMux.

    func GetIQ(n xml.Name, h IQHandler) IQOption
        GetIQ is a shortcut for HandleIQ with the type set to "get".

        For more information, see HandleIQ.

    func HandleIQ(iqType stanza.IQType, n xml.Name, h IQHandler) IQOption
        HandleIQ returns an option that matches the IQ payload by XML name and IQ
        type.

    func HandleIQFunc(iqType stanza.IQType, n xml.Name, h IQHandlerFunc) IQOption
        HandleIQFunc returns an option that matches the IQ payload by XML name and
        IQ type.

    func SetIQ(n xml.Name, h IQHandler) IQOption
        SetIQ is a shortcut for HandleIQ with the type set to "set".

        For more information, see HandleIQ.

## Open Questions

 - What should we do if an `IQMux` is registered on a top level element other
   than IQ?
 - If an error or result type IQ response is sent after an IQ times out and is
   no longer handled by the server, should the IQ mux handle those too (and what
   should the behavior be)?
