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

The proposed API creates three new types and seven new functions that would need
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

        IQs are matched by the type and the XML name of their first child element
        (if any). If either the namespace or the localname is left off, any
        namespace or localname will be matched. Full XML names take precedence,
        followed by wildcard localnames, followed by wildcard namespaces.

        Unlike get and set type IQs, result IQs may have no child element, and error
        IQs may have more than one child element. Because of this it is normally
        adviseable to register handlers for type Error without any filter on the
        child element since we cannot guarantee what child token will come first and
        be matched against. Similarly, for IQs of type result, it is important to
        note that the start element passed to the handler may be nil, meaning that
        there is no child element.

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

    func ErrorIQ(h IQHandler) IQOption
        ErrorIQ is a shortcut for HandleIQ with the type set to "error" and a
        wildcard XML name.

        This differs from the other IQ types because error IQs may contain one or
        more child elements and we cannot guarantee the order of child elements and
        therefore won't know which element to match on. Instead it is normally wise
        to register a handler for all error type IQs and then skip or handle
        unnecessary payloads until we find the error itself.

    func GetIQ(n xml.Name, h IQHandler) IQOption
        GetIQ is a shortcut for HandleIQ with the type set to "get".

    func HandleIQ(iqType stanza.IQType, n xml.Name, h IQHandler) IQOption
        HandleIQ returns an option that matches the IQ payload by XML name and IQ
        type. For readability, users may want to use the GetIQ, SetIQ, ErrorIQ, and
        ResultIQ shortcuts instead.

        For more details, see the documentation on IQMux.

    func HandleIQFunc(iqType stanza.IQType, n xml.Name, h IQHandlerFunc) IQOption
        HandleIQFunc returns an option that matches the IQ payload by XML name and
        IQ type.

    func ResultIQ(n xml.Name, h IQHandler) IQOption
        ResultIQ is a shortcut for HandleIQ with the type set to "result".

        Unlike IQs of type get, set, and error, result type IQs may or may not
        contain a payload. Because of this it is important to check whether the
        start element is nil in handlers meant to handle result type IQs.

    func SetIQ(n xml.Name, h IQHandler) IQOption
        SetIQ is a shortcut for HandleIQ with the type set to "set".

## Open Questions

 - What should we do if an `IQMux` is registered on a top level element other
   than IQ?
