// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mux

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// IQHandler responds to an IQ stanza.
type IQHandler interface {
	HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error
}

// The IQHandlerFunc type is an adapter to allow the use of ordinary functions
// as IQ handlers. If f is a function with the appropriate signature,
// IQHandlerFunc(f) is an IQHandler that calls f.
type IQHandlerFunc func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error

// HandleIQ calls f(iq, t, start).
func (f IQHandlerFunc) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return f(iq, t, start)
}

type patternKey struct {
	xml.Name
	Type stanza.IQType
}

// IQMux is an XMPP multiplexer meant for handling IQ payloads.
//
// IQs are matched by the type and the XML name of their first child element (if
// any).
// If either the namespace or the localname is left off, any namespace or
// localname will be matched.
// Full XML names take precedence, followed by wildcard localnames, followed by
// wildcard namespaces.
//
// Unlike get and set type IQs, result IQs may have no child element, and error
// IQs may have more than one child element.
// Because of this it is normally adviseable to register handlers for type Error
// without any filter on the child element since we cannot guarantee what child
// token will come first and be matched against.
// Similarly, for IQs of type result, it is important to note that the start
// element passed to the handler may be nil, meaning that there is no child
// element.
type IQMux struct {
	patterns map[patternKey]IQHandler
}

// NewIQMux allocates and returns a new IQMux.
func NewIQMux(opt ...IQOption) *IQMux {
	m := &IQMux{}
	for _, o := range opt {
		o(m)
	}
	return m
}

// Handler returns the handler to use for an IQ payload with the given name and
// type.
// If no handler exists, a default handler is returned (h is always non-nil).
func (m *IQMux) Handler(iqType stanza.IQType, name xml.Name) (h IQHandler, ok bool) {
	pattern := patternKey{Name: name, Type: iqType}
	h = m.patterns[pattern]
	if h != nil {
		return h, true
	}

	n := name
	n.Space = ""
	pattern.Name = n
	h = m.patterns[pattern]
	if h != nil {
		return h, true
	}

	n = name
	n.Local = ""
	pattern.Name = n
	h = m.patterns[pattern]
	if h != nil {
		return h, true
	}

	pattern.Name = xml.Name{}
	h = m.patterns[pattern]
	if h != nil {
		return h, true
	}

	return IQHandlerFunc(iqFallback), false
}

func getPayload(t xmlstream.TokenReadEncoder, start *xml.StartElement) (stanza.IQ, *xml.StartElement, error) {
	iq, err := newIQFromStart(start)
	if err != nil {
		return iq, nil, err
	}

	tok, err := t.Token()
	if err != nil {
		return iq, nil, err
	}
	// If this is a result type IQ (or a malformed IQ) there may be no payload, so
	// make sure start is nil.
	payloadStart, ok := tok.(xml.StartElement)
	start = &payloadStart
	if !ok {
		start = nil
	}
	return iq, start, nil
}

// HandleXMPP dispatches the IQ to the handler whose pattern most closely
// matches start.Name.
func (m *IQMux) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	iq, start, err := getPayload(t, start)
	if err != nil {
		return err
	}

	h, _ := m.Handler(iq.Type, start.Name)
	return h.HandleIQ(iq, t, start)
}

// IQOption configures an IQMux.
type IQOption func(m *IQMux)

// HandleIQ returns an option that matches the IQ payload by XML name and IQ
// type.
// For readability, users may want to use the GetIQ, SetIQ, ErrorIQ, and
// ResultIQ shortcuts instead.
//
// For more details, see the documentation on IQMux.
func HandleIQ(iqType stanza.IQType, n xml.Name, h IQHandler) IQOption {
	return func(m *IQMux) {
		if h == nil {
			panic("mux: nil handler")
		}
		pattern := patternKey{Name: n, Type: iqType}
		if _, ok := m.patterns[pattern]; ok {
			panic("mux: multiple registrations for {" + pattern.Space + "}" + pattern.Local)
		}
		if m.patterns == nil {
			m.patterns = make(map[patternKey]IQHandler)
		}
		m.patterns[pattern] = h
	}
}

// GetIQ is a shortcut for HandleIQ with the type set to "get".
func GetIQ(n xml.Name, h IQHandler) IQOption {
	return HandleIQ(stanza.GetIQ, n, h)
}

// GetIQFunc is a shortcut for HandleIQFunc with the type set to "get".
func GetIQFunc(n xml.Name, h IQHandlerFunc) IQOption {
	return GetIQ(n, h)
}

// SetIQ is a shortcut for HandleIQ with the type set to "set".
func SetIQ(n xml.Name, h IQHandler) IQOption {
	return HandleIQ(stanza.SetIQ, n, h)
}

// SetIQFunc is a shortcut for HandleIQ with the type set to "set".
func SetIQFunc(n xml.Name, h IQHandlerFunc) IQOption {
	return SetIQ(n, h)
}

// ErrorIQ is a shortcut for HandleIQ with the type set to "error" and a
// wildcard XML name.
//
// This differs from the other IQ types because error IQs may contain one or
// more child elements and we cannot guarantee the order of child elements and
// therefore won't know which element to match on.
// Instead it is normally wise to register a handler for all error type IQs and
// then skip or handle unnecessary payloads until we find the error itself.
func ErrorIQ(h IQHandler) IQOption {
	return HandleIQ(stanza.ErrorIQ, xml.Name{}, h)
}

// ErrorIQFunc is a shortcut for HandleIQFunc with the type set to "error" and a
// wildcard XML name.
//
// For more information, see ErrorIQ.
func ErrorIQFunc(h IQHandlerFunc) IQOption {
	return ErrorIQ(h)
}

// ResultIQ is a shortcut for HandleIQ with the type set to "result".
//
// Unlike IQs of type get, set, and error, result type IQs may or may not
// contain a payload.
// Because of this it is important to check whether the start element is nil in
// handlers meant to handle result type IQs.
func ResultIQ(n xml.Name, h IQHandler) IQOption {
	return HandleIQ(stanza.ResultIQ, n, h)
}

// ResultIQFunc is a shortcut for HandleIQFunc with the type set to "result".
//
// For more information, see ResultIQ.
func ResultIQFunc(n xml.Name, h IQHandlerFunc) IQOption {
	return ResultIQ(n, h)
}

// HandleIQFunc returns an option that matches the IQ payload by XML name and IQ
// type.
func HandleIQFunc(iqType stanza.IQType, n xml.Name, h IQHandlerFunc) IQOption {
	return HandleIQ(iqType, n, h)
}

// newIQFromStart takes a start element and returns an IQ.
func newIQFromStart(start *xml.StartElement) (stanza.IQ, error) {
	iq := stanza.IQ{}
	var err error
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			if a.Name.Space != "" {
				continue
			}
			iq.ID = a.Value
		case "to":
			if a.Name.Space != "" {
				continue
			}
			iq.To, err = jid.Parse(a.Value)
			if err != nil {
				return iq, err
			}
		case "from":
			if a.Name.Space != "" {
				continue
			}
			iq.From, err = jid.Parse(a.Value)
			if err != nil {
				return iq, err
			}
		case "lang":
			if a.Name.Space != ns.XML {
				continue
			}
			iq.Lang = a.Value
		case "type":
			if a.Name.Space != "" {
				continue
			}
			iq.Type = stanza.IQType(a.Value)
		}
	}
	return iq, nil
}

func iqFallback(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if iq.Type == stanza.ErrorIQ {
		return nil
	}

	iq.To, iq.From = iq.From, iq.To
	iq.Type = "error"

	e := stanza.Error{
		Type:      stanza.Cancel,
		Condition: stanza.ServiceUnavailable,
	}
	_, err := xmlstream.Copy(t, stanza.WrapIQ(
		iq,
		e.TokenReader(),
	))
	return err
}
