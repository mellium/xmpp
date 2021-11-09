// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/stanza"
)

func isMessageEmptySpace(name xml.Name) bool {
	return name.Local == "message" && (name.Space == "" || name.Space == stanza.NSClient || name.Space == stanza.NSServer)
}

// SendMessage is like Send except that it returns an error if the first token
// read from the input is not a message start token and blocks until an error
// response is received or the context times out.
// Messages are generally fire-and-forget meaning that the success behavior of
// SendMessage is to time out and that methods such as Send should normally be
// used instead.
// It is thus mainly for use by extensions that define an extension-namespaced
// success response to a message being sent and need a mechanism to track
// general error responses without handling every single message sent through
// the session to see if it has a matching ID.
//
// SendMessage is safe for concurrent use by multiple goroutines.
func (s *Session) SendMessage(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error) {
	tok, err := r.Token()
	if err != nil {
		return nil, err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil, fmt.Errorf("expected IQ start element, got %T", tok)
	}
	if !isMessageEmptySpace(start.Name) {
		return nil, fmt.Errorf("expected start element to be a message")
	}

	// If there's no ID, add one.
	idx, _, id, typ := getIDTyp(start.Attr)
	if idx == -1 {
		idx = len(start.Attr)
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: ""})
	}
	if id == "" {
		id = attr.RandomID()
		start.Attr[idx].Value = id
	}

	// If this is an error message we don't ever expect a response, so just send
	// it normally.
	if typ == string(stanza.ErrorMessage) {
		return nil, s.SendElement(ctx, xmlstream.Inner(r), start)
	}

	return s.sendResp(ctx, id, xmlstream.Inner(r), start)
}

// SendMessageElement is like SendMessage except that it wraps the payload in a
// Message element.
// For more information see SendMessage.
//
// SendMessageElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendMessageElement(ctx context.Context, payload xml.TokenReader, msg stanza.Message) (xmlstream.TokenReadCloser, error) {
	return s.SendMessage(ctx, msg.Wrap(payload))
}

// EncodeMessage is like Encode except that it returns an error if v does not
// marshal to an Message stanza and like SendMessage it blocks until an error
// response is received or the context times out.
// For more information see SendMessage.
//
// EncodeMessage is safe for concurrent use by multiple goroutines.
func (s *Session) EncodeMessage(ctx context.Context, v interface{}) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(v)
	if err != nil {
		return nil, err
	}
	return s.SendMessage(ctx, r)
}

// EncodeMessageElement is like EncodeMessage except that it wraps the payload
// in a Message element.
// For more information see SendMessage.
//
// EncodeMessageElement is safe for concurrent use by multiple goroutines.
func (s *Session) EncodeMessageElement(ctx context.Context, payload interface{}, msg stanza.Message) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(payload)
	if err != nil {
		return nil, err
	}
	return s.SendMessageElement(ctx, r, msg)
}
