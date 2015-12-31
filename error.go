// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

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

	InvalidFrom            = StreamError{Err: "invalid-from"}
	InvalidNamespace       = StreamError{Err: "invalid-namespace"}
	InvalidXML             = StreamError{Err: "invalid-xml"}
	NotAuthorized          = StreamError{Err: "not-authorized"}
	NotWellFormed          = StreamError{Err: "not-well-formed"}
	PolicyViolation        = StreamError{Err: "policy-violation"}
	RemoteConnectionFailed = StreamError{Err: "remote-connection-failed"}
	Reset                  = StreamError{Err: "reset"}
	ResourceConstraint     = StreamError{Err: "resource-constraint"}
	RestrictedXML          = StreamError{Err: "restricted-xml"}
	// SeeOtherHost           = StreamError{"see-other-host"}
)

// Returns a new SeeOtherHostError with the given network address as the host.
// If the address appears to be a raw IPv6 address (eg. "::1"), the error wraps
// it in brackets ("[::1]").
func NewSeeOtherHostError(addr net.Addr) StreamError {
	var cdata string

	// If the address looks like an IPv6 literal, wrap it in []
	if ip := net.ParseIP(addr.String()); ip != nil && ip.To4() == nil && ip.To16() != nil {
		cdata = "[" + addr.String() + "]"
	} else {
		cdata = addr.String()
	}

	return StreamError{"see-other-host", []byte(cdata)}
}

// A StreamError represents an unrecoverable stream-level error.
type StreamError struct {
	Err      string
	CharData []byte
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
			CharData []byte `xml:",chardata"`
		} `xml:",any"`
	}{}
	err := d.DecodeElement(&se, &start)
	if err != nil {
		return err
	}
	s.Err = se.Err.XMLName.Local
	s.CharData = se.Err.CharData
	return nil
}

// MarshalXML satisfies the xml package's Marshaler interface and allows
// StreamError's to be correctly marshaled back into XML.
func (s StreamError) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(
		struct {
			Err struct {
				XMLName  xml.Name
				CharData []byte `xml:",chardata"`
			}
		}{
			struct {
				XMLName  xml.Name
				CharData []byte `xml:",chardata"`
			}{
				XMLName:  xml.Name{"urn:ietf:params:xml:ns:xmpp-streams", s.Err},
				CharData: s.CharData,
			},
		},
		xml.StartElement{
			xml.Name{"", "stream:error"},
			[]xml.Attr{},
		},
	)
	return nil
}
