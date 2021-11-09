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

func isPresenceEmptySpace(name xml.Name) bool {
	return name.Local == "presence" && (name.Space == "" || name.Space == stanza.NSClient || name.Space == stanza.NSServer)
}

// SendPresence is like Send except that it returns an error if the first token
// read from the input is not a presence start token and blocks until an error
// response is received or the context times out.
// Presences are generally fire-and-forget meaning that the success behavior of
// SendPresence is to time out and that methods such as Send should normally be
// used instead.
// It is thus mainly for use by extensions that define an extension-namespaced
// success response to a presence being sent and need a mechanism to track
// general error responses without handling every single presence sent through
// the session to see if it has a matching ID.
//
// SendPresence is safe for concurrent use by multiple goroutines.
func (s *Session) SendPresence(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error) {
	tok, err := r.Token()
	if err != nil {
		return nil, err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil, fmt.Errorf("expected IQ start element, got %T", tok)
	}
	if !isPresenceEmptySpace(start.Name) {
		return nil, fmt.Errorf("expected start element to be a presence")
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

	// If this is an error presence we don't ever expect a response, so just send
	// it normally.
	if typ == string(stanza.ErrorPresence) {
		return nil, s.SendElement(ctx, xmlstream.Inner(r), start)
	}

	return s.sendResp(ctx, id, xmlstream.Inner(r), start)
}

// SendPresenceElement is like SendPresence except that it wraps the payload in
// a Presence element.
// For more information see SendPresence.
//
// SendPresenceElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendPresenceElement(ctx context.Context, payload xml.TokenReader, msg stanza.Presence) (xmlstream.TokenReadCloser, error) {
	return s.SendPresence(ctx, msg.Wrap(payload))
}

// EncodePresence is like Encode except that it returns an error if v does not
// marshal to an Presence stanza and like SendPresence it blocks until an error
// response is received or the context times out.
// For more information see SendPresence.
//
// EncodePresence is safe for concurrent use by multiple goroutines.
func (s *Session) EncodePresence(ctx context.Context, v interface{}) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(v)
	if err != nil {
		return nil, err
	}
	return s.SendPresence(ctx, r)
}

// EncodePresenceElement is like EncodePresence except that it wraps the payload
// in a Presence element.
// For more information see SendPresence.
//
// EncodePresenceElement is safe for concurrent use by multiple goroutines.
func (s *Session) EncodePresenceElement(ctx context.Context, payload interface{}, msg stanza.Presence) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(payload)
	if err != nil {
		return nil, err
	}
	return s.SendPresenceElement(ctx, r, msg)
}
