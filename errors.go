// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"strings"

	"golang.org/x/text/language"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ns"
)

type errorType int

const (
	// An error with type Auth indicates that an operation should be retried after
	// providing credentials.
	Auth errorType = iota

	// An error with type Cancel indicates that the error cannot be remedied and
	// the operation should not be retried.
	Cancel

	// An error with type Continue indicates that the operation can proceed (the
	// condition was only a warning).
	Continue

	// An error with type Modify indicates that the operation can be retried after
	// changing the data sent.
	Modify

	// An error with type Wait is temporary and may be retried after waiting.
	Wait
)

func (t errorType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.ToLower(t.String())}, nil
}

func (t *errorType) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "auth":
		*t = Auth
	case "continue":
		*t = Continue
	case "modify":
		*t = Modify
	case "wait":
		*t = Wait
	default: // case "cancel":
		*t = Cancel
	}
	return nil
}

// condition represents a stanza error condition that can be encapsulated by an
// <error/> element.
type condition string

// A list of stanza error conditions defined in RFC 6120 §8.3.3
const (
	BadRequest            condition = "bad-request"
	Conflict              condition = "conflict"
	FeatureNotImplemented condition = "feature-not-implemented"
	Forbidden             condition = "forbidden"
	Gone                  condition = "gone"
	InternalServerError   condition = "internal-server-error"
	ItemNotFound          condition = "item-not-found"
	JIDMalformed          condition = "jid-malformed"
	NotAcceptable         condition = "not-acceptable"
	NotAllowed            condition = "not-allowed"
	NotAuthorized         condition = "not-authorized"
	PolicyViolation       condition = "policy-violation"
	RecipientUnavailable  condition = "recipient-unavailable"
	Redirect              condition = "redirect"
	RegistrationRequired  condition = "registration-required"
	RemoteServerNotFound  condition = "remote-server-not-found"
	RemoteServerTimeout   condition = "remote-server-timeout"
	ResourceConstraint    condition = "resource-constraint"
	ServiceUnavailable    condition = "service-unavailable"
	SubscriptionRequired  condition = "subscription-required"
	UndefinedCondition    condition = "undefined-condition"
	UnexpectedRequest     condition = "unexpected-request"
)

// StanzaError is an implementation of error intended to be marshalable and
// unmarshalable as XML.
type StanzaError struct {
	XMLName   xml.Name
	By        *jid.JID
	Type      errorType
	Condition condition
	Lang      language.Tag
	Text      string
}

// Error satisfies the error interface and returns the text if set, or the
// condition otherwise.
func (e StanzaError) Error() string {
	if e.Text != "" {
		return e.Text
	}
	return string(e.Condition)
}

func (se StanzaError) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start = xml.StartElement{
		Name: xml.Name{Space: ``, Local: "error"},
		Attr: []xml.Attr{},
	}
	typattr, err := se.Type.MarshalXMLAttr(xml.Name{Space: "", Local: "type"})
	if err != nil {
		return err
	}
	start.Attr = append(start.Attr, typattr)
	if se.By != nil {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Space: "", Local: "by"}, Value: ""})
	}
	if err = e.EncodeToken(start); err != nil {
		return err
	}
	condition := xml.StartElement{
		Name: xml.Name{Space: ns.Stanza, Local: string(se.Condition)},
	}
	if err = e.EncodeToken(condition); err != nil {
		return err
	}
	if err = e.EncodeToken(condition.End()); err != nil {
		return err
	}
	if se.Text != "" {
		text := xml.StartElement{
			Name: xml.Name{Space: ns.Stanza, Local: "text"},
			Attr: []xml.Attr{
				xml.Attr{
					Name:  xml.Name{Space: ns.XML, Local: "lang"},
					Value: se.Lang.String(),
				},
			},
		}
		if err = e.EncodeToken(text); err != nil {
			return err
		}
		if err = e.EncodeToken(xml.CharData(se.Text)); err != nil {
			return err
		}
		if err = e.EncodeToken(text.End()); err != nil {
			return err
		}
	}
	e.EncodeToken(start.End())
	return nil
}
