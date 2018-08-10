// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

// ErrorType is the type of an stanza error payloads.
// It should normally be one of the constants defined in this package.
type ErrorType string

const (
	// Cancel indicates that the error cannot be remedied and the operation should
	// not be retried.
	Cancel ErrorType = "cancel"

	// Auth indicates that an operation should be retried after providing
	// credentials.
	Auth ErrorType = "auth"

	// Continue indicates that the operation can proceed (the condition was only a
	// warning).
	Continue ErrorType = "continue"

	// Modify indicates that the operation can be retried after changing the data
	// sent.
	Modify ErrorType = "modify"

	// Wait is indicates that an error is temporary and may be retried.
	Wait ErrorType = "wait"
)

// Condition represents a more specific stanza error condition that can be
// encapsulated by an <error/> element.
type Condition string

// A list of stanza error conditions defined in RFC 6120 §8.3.3
const (
	// The sender has sent a stanza containing XML that does not conform to
	// the appropriate schema or that cannot be processed (e.g., an IQ
	// stanza that includes an unrecognized value of the 'type' attribute
	// or an element that is qualified by a recognized namespace but that
	// violates the defined syntax for the element); the associated error
	// type SHOULD be "modify".
	BadRequest Condition = "bad-request"

	// Access cannot be granted because an existing resource exists with the
	// same name or address; the associated error type SHOULD be "cancel".
	Conflict Condition = "conflict"

	// The feature represented in the XML stanza is not implemented by the
	// intended recipient or an intermediate server and therefore the stanza
	// cannot be processed (e.g., the entity understands the namespace but
	// does not recognize the element name); the associated error type
	// SHOULD be "cancel" or "modify".
	FeatureNotImplemented Condition = "feature-not-implemented"

	// The requesting entity does not possess the necessary permissions to
	// perform an action that only certain authorized roles or individuals
	// are allowed to complete (i.e., it typically relates to authorization
	// rather than authentication); the associated error type SHOULD be
	// "auth".
	Forbidden Condition = "forbidden"

	// The recipient or server can no longer be contacted at this address,
	// typically on a permanent basis (as opposed to the <redirect/> error
	// condition, which is used for temporary addressing failures); the
	// associated error type SHOULD be "cancel" and the error stanza SHOULD
	// include a new address (if available) as the XML character data of the
	// <gone/> element (which MUST be a Uniform Resource Identifier [URI] or
	// Internationalized Resource Identifier [IRI] at which the entity can
	// be contacted, typically an XMPP IRI as specified in RFC 5122).
	Gone Condition = "gone"

	// The server has experienced a misconfiguration or other internal error
	// that prevents it from processing the stanza; the associated error
	// type SHOULD be "cancel".
	InternalServerError Condition = "internal-server-error"

	// The addressed JID or item requested cannot be found; the associated
	// error type SHOULD be "cancel".
	//
	// Security Warning: An application MUST NOT return this error if
	// doing so would provide information about the intended recipient's
	// network availability to an entity that is not authorized to know
	// such information (for a more detailed discussion of presence
	// authorization, refer to the discussion of presence subscriptions
	// in RFC 6121); instead it MUST return a ServiceUnavailable
	// stanza error.
	ItemNotFound Condition = "item-not-found"

	// The sending entity has provided (e.g., during resource binding) or
	// communicated (e.g., in the 'to' address of a stanza) an XMPP address
	// that violates the rules of the mellium.im/xmpp/jid package; the
	// associated error type SHOULD be "modify".
	//
	//  Implementation Note: Enforcement of the format for XMPP localparts
	//  is primarily the responsibility of the service at which the
	//  associated account or entity is located (e.g., the example.com
	//  service is responsible for returning <jid-malformed/> errors
	//  related to all JIDs of the form <localpart@example.com>), whereas
	//  enforcement of the format for XMPP domainparts is primarily the
	//  responsibility of the service that seeks to route a stanza to the
	//  service identified by that domainpart (e.g., the example.org
	//  service is responsible for returning <jid-malformed/> errors
	//  related to stanzas that users of that service have to tried send
	//  to JIDs of the form <localpart@example.com>).  However, any entity
	//  that detects a malformed JID MAY return this error.
	JIDMalformed Condition = "jid-malformed"

	// The recipient or server understands the request but cannot process it
	// because the request does not meet criteria defined by the recipient
	// or server (e.g., a request to subscribe to information that does not
	// simultaneously include configuration parameters needed by the
	// recipient); the associated error type SHOULD be "modify".
	NotAcceptable Condition = "not-acceptable"

	// The recipient or server does not allow any entity to perform the
	// action (e.g., sending to entities at a blacklisted domain); the
	// associated error type SHOULD be "cancel".
	NotAllowed Condition = "not-allowed"

	// The sender needs to provide credentials before being allowed to
	// perform the action, or has provided improper credentials (the name
	// "not-authorized", which was borrowed from the "401 Unauthorized"
	// error of HTTP, might lead the reader to think that this condition
	// relates to authorization, but instead it is typically used in
	// relation to authentication); the associated error type SHOULD be
	// "auth".
	NotAuthorized Condition = "not-authorized"

	// The entity has violated some local service policy (e.g., a message
	// contains words that are prohibited by the service) and the server MAY
	// choose to specify the policy in the <text/> element or in an
	// application-specific condition element; the associated error type
	// SHOULD be "modify" or "wait" depending on the policy being violated.
	PolicyViolation Condition = "policy-violation"

	// The intended recipient is temporarily unavailable, undergoing
	// maintenance, etc.; the associated error type SHOULD be "wait".
	//
	// Security Warning: An application MUST NOT return this error if
	// doing so would provide information about the intended recipient's
	// network availability to an entity that is not authorized to know
	// such information (for a more detailed discussion of presence
	// authorization, refer to the discussion of presence subscriptions
	// in RFC 6121); instead it MUST return a ServiceUnavailable stanza
	// error.
	RecipientUnavailable Condition = "recipient-unavailable"

	// The recipient or server is redirecting requests for this information
	// to another entity, typically in a temporary fashion (as opposed to
	// the <gone/> error condition, which is used for permanent addressing
	// failures); the associated error type SHOULD be "modify" and the error
	// stanza SHOULD contain the alternate address in the XML character data
	// of the <redirect/> element (which MUST be a URI or IRI with which the
	// sender can communicate, typically an XMPP IRI as specified in
	// RFC 5122).
	//
	// Security Warning: An application receiving a stanza-level redirect
	// SHOULD warn a human user of the redirection attempt and request
	// approval before proceeding to communicate with the entity whose
	// address is contained in the XML character data of the <redirect/>
	// element, because that entity might have a different identity or
	// might enforce different security policies.  The end-to-end
	// authentication or signing of XMPP stanzas could help to mitigate
	// this risk, since it would enable the sender to determine if the
	// entity to which it has been redirected has the same identity as
	// the entity it originally attempted to contact.  An application MAY
	// have a policy of following redirects only if it has authenticated
	// the receiving entity.  In addition, an application SHOULD abort
	// the communication attempt after a certain number of successive
	// redirects (e.g., at least 2 but no more than 5).
	Redirect Condition = "redirect"

	// The requesting entity is not authorized to access the requested
	// service because prior registration is necessary (examples of prior
	// registration include members-only rooms in XMPP multi-user chat
	// [XEP-0045] and gateways to non-XMPP instant messaging services, which
	// traditionally required registration in order to use the gateway
	// [XEP-0100]); the associated error type SHOULD be "auth".
	RegistrationRequired Condition = "registration-required"

	// A remote server or service specified as part or all of the JID of the
	// intended recipient does not exist or cannot be resolved (e.g., there
	// is no _xmpp-server._tcp DNS SRV record, the A or AAAA fallback
	// resolution fails, or A/AAAA lookups succeed but there is no response
	// on the IANA-registered port 5269); the associated error type SHOULD
	// be "cancel".
	RemoteServerNotFound Condition = "remote-server-not-found"

	// A remote server or service specified as part or all of the JID of the
	// intended recipient (or needed to fulfill a request) was resolved but
	// communications could not be established within a reasonable amount of
	// time (e.g., an XML stream cannot be established at the resolved IP
	// address and port, or an XML stream can be established but stream
	// negotiation fails because of problems with TLS, SASL, Server
	// Dialback, etc.); the associated error type SHOULD be "wait" (unless
	// the error is of a more permanent nature, e.g., the remote server is
	// found but it cannot be authenticated or it violates security
	// policies).
	RemoteServerTimeout Condition = "remote-server-timeout"

	// The server or recipient is busy or lacks the system resources
	// necessary to service the request; the associated error type SHOULD be
	// "wait".
	ResourceConstraint Condition = "resource-constraint"

	// The server or recipient does not currently provide the requested
	// service; the associated error type SHOULD be "cancel".
	//
	// Security Warning: An application MUST return a ServiceUnavailable
	// stanza error instead of ItemNotFound or RecipientUnavailable if
	// if sending one of the latter errors would provide information about
	// the intended recipient's network availability to an entity that is
	// not authorized to know such information (for a more detailed discussion
	// of presence authorization, refer to RFC 6121).
	ServiceUnavailable Condition = "service-unavailable"

	// The requesting entity is not authorized to access the requested
	// service because a prior subscription is necessary (examples of prior
	// subscription include authorization to receive presence information as
	// defined in RFC 6121 and opt-in data feeds for XMPP publish-subscribe
	// as defined in [XEP-0060]); the associated error type SHOULD be
	// "auth".
	SubscriptionRequired Condition = "subscription-required"

	// The error condition is not one of those defined by the other
	// conditions in this list; any error type can be associated with this
	// condition, and it SHOULD NOT be used except in conjunction with an
	// application-specific condition.
	UndefinedCondition Condition = "undefined-condition"

	// The recipient or server understood the request but was not expecting
	// it at this time (e.g., the request was out of order); the associated
	// error type SHOULD be "wait" or "modify".
	UnexpectedRequest Condition = "unexpected-request"
)

// Error is an implementation of error intended to be marshalable and
// unmarshalable as XML.
type Error struct {
	XMLName   xml.Name
	By        jid.JID
	Type      ErrorType
	Condition Condition
	Text      map[string]string
}

// Error satisfies the error interface by returning the condition.
func (se Error) Error() string {
	return string(se.Condition)
}

// TokenReader satisfies the xmlstream.Marshaler interface for Error.
func (se Error) TokenReader() xml.TokenReader {
	start := xml.StartElement{
		Name: xml.Name{Space: ``, Local: "error"},
		Attr: []xml.Attr{},
	}
	if string(se.Type) != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: string(se.Type)})
	}
	a, err := se.By.MarshalXMLAttr(xml.Name{Space: "", Local: "by"})
	if err == nil && a.Value != "" {
		start.Attr = append(start.Attr, a)
	}

	var text xml.TokenReader = xmlstream.ReaderFunc(func() (xml.Token, error) {
		return nil, io.EOF
	})
	for lang, data := range se.Text {
		if data == "" {
			continue
		}
		var attrs []xml.Attr
		// xml:lang attribute is optional, don't include it if it's empty.
		if lang != "" {
			attrs = []xml.Attr{{
				Name:  xml.Name{Space: ns.XML, Local: "lang"},
				Value: lang,
			}}
		}
		text = xmlstream.Wrap(
			xmlstream.ReaderFunc(func() (xml.Token, error) {
				return xml.CharData(data), io.EOF
			}),
			xml.StartElement{
				Name: xml.Name{Space: ns.Stanza, Local: "text"},
				Attr: attrs,
			},
		)
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Wrap(
				nil,
				xml.StartElement{
					Name: xml.Name{Space: ns.Stanza, Local: string(se.Condition)},
				},
			),
			text,
		),
		start,
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (se Error) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, se.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface for Error.
func (se Error) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := se.WriteXML(e)
	return err
}

// UnmarshalXML satisfies the xml.Unmarshaler interface for StanzaError.
func (se *Error) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		Condition struct {
			XMLName xml.Name
		} `xml:",any"`
		Type ErrorType `xml:"type,attr"`
		By   jid.JID   `xml:"by,attr"`
		Text []struct {
			Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
			Data string `xml:",chardata"`
		} `xml:"urn:ietf:params:xml:ns:xmpp-stanzas text"`
	}{}
	if err := d.DecodeElement(&decoded, &start); err != nil {
		return err
	}
	se.Type = decoded.Type
	se.By = decoded.By
	if decoded.Condition.XMLName.Space == ns.Stanza {
		se.Condition = Condition(decoded.Condition.XMLName.Local)
	}

	for _, text := range decoded.Text {
		if text.Data == "" {
			continue
		}
		if se.Text == nil {
			se.Text = make(map[string]string)
		}
		se.Text[text.Lang] = text.Data
	}
	return nil
}
