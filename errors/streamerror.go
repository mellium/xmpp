// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package errors

import (
	"encoding/xml"
	"net"
)

// A list of stream errors defined in RFC 6120 §4.9.3
var (
	// BadFormat is used when the entity has sent XML that cannot be processed.
	// This error can be used instead of the more specific XML-related errors,
	// such as <bad-namespace-prefix/>, <invalid-xml/>, <not-well-formed/>,
	// <restricted-xml/>, and <unsupported-encoding/>. However, the more specific
	// errors are RECOMMENDED.
	BadFormat = StreamError{Err: "bad-format"}

	// BadNamespacePrefix is sent when an entity has sent a namespace prefix that
	// is unsupported, or has sent no namespace prefix, on an element that needs
	// such a prefix.
	BadNamespacePrefix = StreamError{Err: "bad-namespace-prefix"}

	// Conflict is sent when the server either (1) is closing the existing stream
	// for this entity because a new stream has been initiated that conflicts with
	// the existing stream, or (2) is refusing a new stream for this entity
	// because allowing the new stream would conflict with an existing stream
	// (e.g., because the server allows only a certain number of connections from
	// the same IP address or allows only one server-to-server stream for a given
	// domain pair as a way of helping to ensure in-order processing.
	Conflict = StreamError{Err: "conflict"}

	// ConnectionTimeout results when one party is closing the stream because it
	// has reason to believe that the other party has permanently lost the ability
	// to communicate over the stream.
	ConnectionTimeout = StreamError{Err: "connection-timeout"}

	// HostGone is sent when the value of the 'to' attribute provided in the
	// initial stream header corresponds to an FQDN that is no longer serviced by
	// the receiving entity.
	HostGone = StreamError{Err: "host-gone"}

	// HostUnknown is sent when the value of the 'to' attribute provided in the
	// initial stream header does not correspond to an FQDN that is serviced by
	// the receiving entity.
	HostUnknown = StreamError{Err: "host-unknown"}

	// ImproperAddressing is used when a stanza sent between two servers lacks a
	// 'to' or 'from' attribute, the 'from' or 'to' attribute has no value, or the
	// value violates the rules for XMPP addresses.
	ImproperAddressing = StreamError{Err: "improper-addressing"}

	// InternalServerError is sent when the server has experienced a
	// misconfiguration or other internal error that prevents it from servicing
	// the stream.
	InternalServerError = StreamError{Err: "internal-server-error"}

	// InvalidFrom is sent when data provided in a 'from' attribute does not match
	// an authorized JID or validated domain as negotiated (1) between two servers
	// using SASL or Server Dialback, or (2) between a client and a server via
	// SASL authentication and resource binding.
	InvalidFrom = StreamError{Err: "invalid-from"}

	// InvalidNamespace may be sent when the stream namespace name is something
	// other than "http://etherx.jabber.org/streams" or the content namespace
	// declared as the default namespace is not supported (e.g., something other
	// than "jabber:client" or "jabber:server").
	InvalidNamespace = StreamError{Err: "invalid-namespace"}

	// InvalidXML may be sent when the entity has sent invalid XML over the stream
	// to a server that performs validation.
	InvalidXML = StreamError{Err: "invalid-xml"}

	// NotAuthorized may be sent when the entity has attempted to send XML stanzas
	// or other outbound data before the stream has been authenticated, or
	// otherwise is not authorized to perform an action related to stream
	// negotiation; the receiving entity MUST NOT process the offending data
	// before sending the stream error.
	NotAuthorized = StreamError{Err: "not-authorized"}

	// NotWellFormed may be sent when the initiating entity has sent XML that
	// violates the well-formedness rules of XML or XML namespaces.
	NotWellFormed = StreamError{Err: "not-well-formed"}

	// PolicyViolation may be sent when an entity has violated some local service
	// policy (e.g., a stanza exceeds a configured size limit).
	PolicyViolation = StreamError{Err: "policy-violation"}

	// RemoteConnectionFailed may be sent when the server is unable to properly
	// connect to a remote entity that is needed for authentication or
	// authorization
	RemoteConnectionFailed = StreamError{Err: "remote-connection-failed"}

	// server is closing the stream because it has new (typically
	// security-critical) features to offer, because the keys or certificates used
	// to establish a secure context for the stream have expired or have been
	// revoked during the life of the stream, because the TLS sequence number has
	// wrapped, etc. Encryption and authentication need to be negotiated again for
	// the new stream (e.g., TLS session resumption cannot be used).
	Reset = StreamError{Err: "reset"}

	// ResourceConstraing may be sent when the server lacks the system resources
	// necessary to service the stream.
	ResourceConstraint = StreamError{Err: "resource-constraint"}

	// RestrictedXML may be sent when the entity has attempted to send restricted
	// XML features such as a comment, processing instruction, DTD subset, or XML
	// entity reference.
	RestrictedXML = StreamError{Err: "restricted-xml"}

	// SystemShutdown may be sent when server is being shut down and all active
	// streams are being closed.
	SystemShutdown = StreamError{Err: "system-shutdown"}

	// UnsupportedEncoding may be sent when initiating entity has encoded the
	// stream in an encoding that is not UTF-8.
	UnsupportedEncoding = StreamError{Err: "unsupported-encoding"}

	// UnsupportedFeature may be sent when receiving entity has advertised a
	// mandatory-to-negotiate stream feature that the initiating entity does not
	// support.
	UnsupportedFeature = StreamError{Err: "unsupported-feature"}

	// UnsupportedStanzaType may be sent when the initiating entity has sent a
	// first-level child of the stream that is not supported by the server, either
	// because the receiving entity does not understand the namespace or because
	// the receiving entity does not understand the element name for the
	// applicable namespace (which might be the content namespace declared as the
	// default namespace).
	UnsupportedStanzaType = StreamError{Err: "unsupported-stanza-type"}

	// UnsupportedVersion may be sent when the 'version' attribute provided by the
	// initiating entity in the stream header specifies a version of XMPP that is
	// not supported by the server.
	UnsupportedVersion = StreamError{Err: "unsupported-version"}
)

// SeeOtherHost returns a new see-other-host error with the given network
// address as the host. If the address appears to be a raw IPv6 address (eg.
// "::1"), the error wraps it in brackets ("[::1]").
func SeeOtherHost(addr net.Addr) StreamError {
	var cdata string

	// If the address looks like an IPv6 literal, wrap it in []
	if ip := net.ParseIP(addr.String()); ip != nil && ip.To4() == nil && ip.To16() != nil {
		cdata = "[" + addr.String() + "]"
	} else {
		cdata = addr.String()
	}

	return StreamError{"see-other-host", []byte(cdata)}
}

// A StreamError represents an unrecoverable stream-level error that may include
// character data or arbitrary inner XML.
type StreamError struct {
	Err      string
	InnerXML []byte
}

// Error satisfies the builtin error interface and returns the name of the
// StreamError. For instance, given the error:
//
//     <stream:error>
//       <restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"/>
//     </stream:error>
//
// Error() would return "restricted-xml".
func (e *StreamError) Error() string {
	return e.Err
}

// UnmarshalXML satisfies the xml package's Unmarshaler interface and allows
// StreamError's to be correctly unmarshaled from XML.
func (s *StreamError) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
	s.InnerXML = se.Err.InnerXML
	return nil
}

// MarshalXML satisfies the xml package's Marshaler interface and allows
// StreamError's to be correctly marshaled back into XML.
func (s StreamError) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(
		struct {
			Err struct {
				XMLName  xml.Name
				InnerXML []byte `xml:",innerxml"`
			}
		}{
			struct {
				XMLName  xml.Name
				InnerXML []byte `xml:",innerxml"`
			}{
				XMLName:  xml.Name{"urn:ietf:params:xml:ns:xmpp-streams", s.Err},
				InnerXML: s.InnerXML,
			},
		},
		xml.StartElement{
			xml.Name{"", "stream:error"},
			[]xml.Attr{},
		},
	)
	return nil
}
