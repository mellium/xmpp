// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"strings"

	"mellium.im/xmpp/jid"
)

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	From     *jid.JID `xml:"from,attr"`
	To       *jid.JID `xml:"to,attr"`
	ID       string   `xml:"id,attr"`
	Lang     string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	InnerXML []byte   `xml:",innerxml"`
	Type     iqType   `xml:"type,attr"`
	XMLName  xml.Name `xml:"iq"`
}

type iqType int

const (
	// A Get IQ is used to query another entity for information.
	Get iqType = iota

	// A Set IQ is used to provide data to another entity, set new values, and
	// replace existing values.
	Set

	// A Result IQ is sent in response to a successful get or set IQ.
	Result

	// An Error IQ is sent to report that an error occured during the delivery or
	// processing of a get or set IQ.
	Error
)

func (t iqType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.ToLower(t.String())}, nil
}

func (t *iqType) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "get":
		*t = Get
	case "set":
		*t = Set
	case "result":
		*t = Result
	case "error":
		*t = Error
	default:
		// TODO: This should be a stanza error with the bad-request condition.
		return errors.New("bad-request")
	}
	return nil
}

// TODO: Should this be variadic and accept many payloads or many to's?
func (c *Conn) sendIQ(ctx context.Context, to *jid.JID, t iqType, v interface{}) (*IQ, error) {
	panic("xmpp: sendIQ not yet implemented")
}

// Do sends an IQ request and blocks until an IQ response is received. If the
// provided context expires before a response is received, the function unblocks
// and an error is returned. An error is only returned on timeouts or connection
// errors, error resonses to the IQ are expected and returned as normal IQ
// responses (err will still be nil).
//
// If the IQ is not of type Get or Set, panic.
func (c *Conn) Do(ctx context.Context, iq IQ) (resp *IQ, err error) {
	if iq.Type != Get && iq.Type != Set {
		panic("xmpp: Attempted to send non-response IQ with invalid type.")
	}
	return c.sendIQ(ctx, nil, iq.Type, iq)
}

// Set marshals the provided data to XML and then sends it as the payload of a
// "set" IQ stanza. For more information, see the Do function.
func (c *Conn) Set(ctx context.Context, to *jid.JID, payload interface{}) (resp *IQ, err error) {
	return c.sendIQ(ctx, to, Set, payload)
}

// Get marshals the provided data to XML and then sends it as the payload of a
// "get" IQ stanza. For more information see the Do function.
func (c *Conn) Get(ctx context.Context, to *jid.JID, payload interface{}) (resp *IQ, err error) {
	return c.sendIQ(ctx, to, Get, payload)
}
