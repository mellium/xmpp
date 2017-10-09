// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package stream contains XMPP stream errors as defined by RFC 6120 §4.9.
//
// Most people will want to use the facilities of the mellium.im/xmpp package
// and not create stream errors directly.
package stream // import "mellium.im/xmpp/stream"

import (
	"encoding/xml"
	"io"
	"net"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
)

// A list of stream errors defined in RFC 6120 §4.9.3
var (
	// BadFormat is used when the entity has sent XML that cannot be processed.
	// This error can be used instead of the more specific XML-related errors,
	// such as <bad-namespace-prefix/>, <invalid-xml/>, <not-well-formed/>,
	// <restricted-xml/>, and <unsupported-encoding/>. However, the more specific
	// errors are RECOMMENDED.
	BadFormat = Error{Err: "bad-format"}

	// BadNamespacePrefix is sent when an entity has sent a namespace prefix that
	// is unsupported, or has sent no namespace prefix, on an element that needs
	// such a prefix.
	BadNamespacePrefix = Error{Err: "bad-namespace-prefix"}

	// Conflict is sent when the server either (1) is closing the existing stream
	// for this entity because a new stream has been initiated that conflicts with
	// the existing stream, or (2) is refusing a new stream for this entity
	// because allowing the new stream would conflict with an existing stream
	// (e.g., because the server allows only a certain number of connections from
	// the same IP address or allows only one server-to-server stream for a given
	// domain pair as a way of helping to ensure in-order processing.
	Conflict = Error{Err: "conflict"}

	// ConnectionTimeout results when one party is closing the stream because it
	// has reason to believe that the other party has permanently lost the ability
	// to communicate over the stream.
	ConnectionTimeout = Error{Err: "connection-timeout"}

	// HostGone is sent when the value of the 'to' attribute provided in the
	// initial stream header corresponds to an FQDN that is no longer serviced by
	// the receiving entity.
	HostGone = Error{Err: "host-gone"}

	// HostUnknown is sent when the value of the 'to' attribute provided in the
	// initial stream header does not correspond to an FQDN that is serviced by
	// the receiving entity.
	HostUnknown = Error{Err: "host-unknown"}

	// ImproperAddressing is used when a stanza sent between two servers lacks a
	// 'to' or 'from' attribute, the 'from' or 'to' attribute has no value, or the
	// value violates the rules for XMPP addresses.
	ImproperAddressing = Error{Err: "improper-addressing"}

	// InternalServerError is sent when the server has experienced a
	// misconfiguration or other internal error that prevents it from servicing
	// the stream.
	InternalServerError = Error{Err: "internal-server-error"}

	// InvalidFrom is sent when data provided in a 'from' attribute does not match
	// an authorized JID or validated domain as negotiated (1) between two servers
	// using SASL or Server Dialback, or (2) between a client and a server via
	// SASL authentication and resource binding.
	InvalidFrom = Error{Err: "invalid-from"}

	// InvalidNamespace may be sent when the stream namespace name is something
	// other than "http://etherx.jabber.org/streams" or the content namespace
	// declared as the default namespace is not supported (e.g., something other
	// than "jabber:client" or "jabber:server").
	InvalidNamespace = Error{Err: "invalid-namespace"}

	// InvalidXML may be sent when the entity has sent invalid XML over the stream
	// to a server that performs validation.
	InvalidXML = Error{Err: "invalid-xml"}

	// NotAuthorized may be sent when the entity has attempted to send XML stanzas
	// or other outbound data before the stream has been authenticated, or
	// otherwise is not authorized to perform an action related to stream
	// negotiation; the receiving entity MUST NOT process the offending data
	// before sending the stream error.
	NotAuthorized = Error{Err: "not-authorized"}

	// NotWellFormed may be sent when the initiating entity has sent XML that
	// violates the well-formedness rules of XML or XML namespaces.
	NotWellFormed = Error{Err: "not-well-formed"}

	// PolicyViolation may be sent when an entity has violated some local service
	// policy (e.g., a stanza exceeds a configured size limit).
	PolicyViolation = Error{Err: "policy-violation"}

	// RemoteConnectionFailed may be sent when the server is unable to properly
	// connect to a remote entity that is needed for authentication or
	// authorization
	RemoteConnectionFailed = Error{Err: "remote-connection-failed"}

	// server is closing the stream because it has new (typically
	// security-critical) features to offer, because the keys or certificates used
	// to establish a secure context for the stream have expired or have been
	// revoked during the life of the stream, because the TLS sequence number has
	// wrapped, etc. Encryption and authentication need to be negotiated again for
	// the new stream (e.g., TLS session resumption cannot be used).
	Reset = Error{Err: "reset"}

	// ResourceConstraing may be sent when the server lacks the system resources
	// necessary to service the stream.
	ResourceConstraint = Error{Err: "resource-constraint"}

	// RestrictedXML may be sent when the entity has attempted to send restricted
	// XML features such as a comment, processing instruction, DTD subset, or XML
	// entity reference.
	RestrictedXML = Error{Err: "restricted-xml"}

	// SystemShutdown may be sent when server is being shut down and all active
	// streams are being closed.
	SystemShutdown = Error{Err: "system-shutdown"}

	// UndefinedCondition may be sent when the error condition is not one of those
	// defined by the other conditions in this list; this error condition should
	// be used in conjunction with an application-specific condition.
	UndefinedCondition = Error{Err: "undefined-condition"}

	// UnsupportedEncoding may be sent when initiating entity has encoded the
	// stream in an encoding that is not UTF-8.
	UnsupportedEncoding = Error{Err: "unsupported-encoding"}

	// UnsupportedFeature may be sent when receiving entity has advertised a
	// mandatory-to-negotiate stream feature that the initiating entity does not
	// support.
	UnsupportedFeature = Error{Err: "unsupported-feature"}

	// UnsupportedStanzaType may be sent when the initiating entity has sent a
	// first-level child of the stream that is not supported by the server, either
	// because the receiving entity does not understand the namespace or because
	// the receiving entity does not understand the element name for the
	// applicable namespace (which might be the content namespace declared as the
	// default namespace).
	UnsupportedStanzaType = Error{Err: "unsupported-stanza-type"}

	// UnsupportedVersion may be sent when the 'version' attribute provided by the
	// initiating entity in the stream header specifies a version of XMPP that is
	// not supported by the server.
	UnsupportedVersion = Error{Err: "unsupported-version"}
)

// SeeOtherHostError returns a new see-other-host error with the given network
// address as the host. If the address appears to be a raw IPv6 address (eg.
// "::1"), the error wraps it in brackets ("[::1]").
func SeeOtherHostError(addr net.Addr, payload xmlstream.TokenReader) Error {
	var cdata string

	// If the address looks like an IPv6 literal, wrap it in []
	if ip := net.ParseIP(addr.String()); ip != nil && ip.To4() == nil && ip.To16() != nil {
		cdata = "[" + addr.String() + "]"
	} else {
		cdata = addr.String()
	}

	if payload != nil {
		payload = xmlstream.MultiReader(
			xmlstream.ReaderFunc(func() (xml.Token, error) {
				return xml.CharData(cdata), io.EOF
			}),
			payload,
		)
	} else {
		payload = xmlstream.ReaderFunc(func() (xml.Token, error) {
			return xml.CharData(cdata), io.EOF
		})
	}

	return Error{Err: "see-other-host", innerXML: payload}
}

// A Error represents an unrecoverable stream-level error that may include
// character data or arbitrary inner XML.
type Error struct {
	Err string

	innerXML xmlstream.TokenReader `xml:"-"`
}

// Error satisfies the builtin error interface and returns the name of the
// StreamError. For instance, given the error:
//
//     <stream:error>
//       <restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"/>
//     </stream:error>
//
// Error() would return "restricted-xml".
func (s Error) Error() string {
	return s.Err
}

// UnmarshalXML satisfies the xml package's Unmarshaler interface and allows
// StreamError's to be correctly unmarshaled from XML.
func (s *Error) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	se := struct {
		XMLName xml.Name
		Err     struct {
			XMLName  xml.Name
			InnerXML []byte `xml:",innerxml"`
		} `xml:",any"`
	}{}
	err := d.DecodeElement(&se, &start)
	if err != nil {
		return err
	}
	s.Err = se.Err.XMLName.Local
	// TODO: s.InnerXML = se.Err.InnerXML
	return nil
}

// MarshalXML satisfies the xml package's Marshaler interface and allows
// StreamError's to be correctly marshaled back into XML.
func (s Error) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	return s.WriteXML(e, xml.StartElement{})
}

// WriteXML satisfies the xmlstream.Marshaler interface.
// It is like MarshalXML except it writes tokens to w.
func (s Error) WriteXML(w xmlstream.TokenWriter, _ xml.StartElement) error {
	_, err := xmlstream.Copy(w, s.TokenReader(nil))
	if err != nil {
		return err
	}
	return w.Flush()
}

// TokenReader returns a new xmlstream.TokenReader that returns an encoding of
// the error.
func (s Error) TokenReader(payload xmlstream.TokenReader) xmlstream.TokenReader {
	inner := xmlstream.Wrap(s.innerXML, xml.StartElement{Name: xml.Name{Local: s.Err, Space: ns.Streams}})
	if payload != nil {
		inner = xmlstream.MultiReader(
			inner,
			payload,
		)
	}
	return xmlstream.Wrap(
		inner,
		xml.StartElement{
			Name: xml.Name{Local: "error", Space: ns.Stream},
		},
	)
}
