// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"strings"

	"golang.org/x/text/language"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

type errorType int

const (
	// Cancel indicates that the error cannot be remedied and the operation should
	// not be retried.
	Cancel errorType = iota

	// Auth indicates that an operation should be retried after providing
	// credentials.
	Auth

	// Continue indicates that the operation can proceed (the condition was only a
	// warning).
	Continue

	// Modify indicates that the operation can be retried after changing the data
	// sent.
	Modify

	// Wait is indicates that an error is temporary and may be retried.
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

// Error is an implementation of error intended to be marshalable and
// unmarshalable as XML.
type Error struct {
	XMLName   xml.Name
	By        *jid.JID
	Type      errorType
	Condition condition
	Lang      language.Tag
	Text      string
}

// Error satisfies the error interface and returns the text if set, or the
// condition otherwise.
func (se Error) Error() string {
	if se.Text != "" {
		return se.Text
	}
	return string(se.Condition)
}

// MarshalXML satisfies the xml.Marshaler interface for StanzaError.
func (se Error) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	start = xml.StartElement{
		Name: xml.Name{Space: ``, Local: "error"},
		Attr: []xml.Attr{},
	}
	typattr, _ := se.Type.MarshalXMLAttr(xml.Name{Space: "", Local: "type"})
	start.Attr = append(start.Attr, typattr)
	if se.By != nil {
		a, _ := se.By.MarshalXMLAttr(xml.Name{Space: "", Local: "by"})
		start.Attr = append(start.Attr, a)
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
				{
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

// UnmarshalXML satisfies the xml.Unmarshaler interface for StanzaError.
func (se *Error) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		Condition struct {
			XMLName xml.Name
		} `xml:",any"`
		Type errorType `xml:"type,attr"`
		By   *jid.JID  `xml:"by,attr"`
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
	// TODO: Oh god why… maybe I should transform the String() output instead.
	switch decoded.Condition.XMLName.Local {
	case "bad-request":
		se.Condition = BadRequest
	case "conflict":
		se.Condition = Conflict
	case "feature-not-implemented":
		se.Condition = FeatureNotImplemented
	case "forbidden":
		se.Condition = Forbidden
	case "gone":
		se.Condition = Gone
	case "internal-server-error":
		se.Condition = InternalServerError
	case "item-not-found":
		se.Condition = ItemNotFound
	case "jid-malformed":
		se.Condition = JIDMalformed
	case "not-acceptable":
		se.Condition = NotAcceptable
	case "not-allowed":
		se.Condition = NotAllowed
	case "not-authorized":
		se.Condition = NotAuthorized
	case "policy-violation":
		se.Condition = PolicyViolation
	case "recipient-unavailable":
		se.Condition = RecipientUnavailable
	case "redirect":
		se.Condition = Redirect
	case "registration-required":
		se.Condition = RegistrationRequired
	case "remote-server-not-found":
		se.Condition = RemoteServerNotFound
	case "remote-server-timeout":
		se.Condition = RemoteServerTimeout
	case "resource-constraint":
		se.Condition = ResourceConstraint
	case "service-unavailable":
		se.Condition = ServiceUnavailable
	case "subscription-required":
		se.Condition = SubscriptionRequired
	case "undefined-condition":
		se.Condition = UndefinedCondition
	case "unexpected-request":
		se.Condition = UnexpectedRequest
	default:
		if decoded.Condition.XMLName.Space == ns.Stanza {
			se.Condition = condition(decoded.Condition.XMLName.Local)
		}
	}

	// TODO: Dedup this (and probably a lot of other stuff) with the saslerr
	// logic.
	tags := make([]language.Tag, 0, len(decoded.Text))
	data := make(map[language.Tag]string)
	for _, text := range decoded.Text {
		tag, err := language.Parse(text.Lang)
		if err != nil {
			continue
		}
		tags = append(tags, tag)
		data[tag] = text.Data
	}
	tag, _, _ := language.NewMatcher(tags).Match(se.Lang)
	se.Lang = tag
	se.Text, _ = data[tag]
	return nil
}
