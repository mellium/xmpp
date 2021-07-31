// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/marshal"
	"mellium.im/xmpp/stanza"
)

// SendIQ is like Send except that it returns an error if the first token read
// from the stream is not an Info/Query (IQ) start element and blocks until a
// response is received.
//
// If the input stream is not being processed (a call to Serve is not running),
// SendIQ will never receive a response and will block until the provided
// context is canceled.
// If the response is non-nil, it does not need to be consumed in its entirety,
// but it must be closed before stream processing will resume.
// If the IQ type does not require a response—ie. it is a result or error IQ,
// meaning that it is a response itself—SendIQElement does not block and the
// response is nil.
//
// If the context is closed before the response is received, SendIQ immediately
// returns the context error.
// Any response received at a later time will not be associated with the
// original request but can still be handled by the Serve handler.
//
// If an error is returned, the response will be nil; the converse is not
// necessarily true.
// SendIQ is safe for concurrent use by multiple goroutines.
func (s *Session) SendIQ(ctx context.Context, r xml.TokenReader) (xmlstream.TokenReadCloser, error) {
	tok, err := r.Token()
	if err != nil {
		return nil, err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil, fmt.Errorf("expected IQ start element, got %T", tok)
	}
	if !isIQEmptySpace(start.Name) {
		return nil, fmt.Errorf("expected start element to be an IQ")
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

	// If this an IQ of type "set" or "get" we expect a response.
	if typ == string(stanza.GetIQ) || typ == string(stanza.SetIQ) {
		return s.sendResp(ctx, id, xmlstream.Inner(r), start)
	}

	// If this is an IQ of type result or error, we don't expect a response so
	// just send it normally.
	return nil, s.SendElement(ctx, xmlstream.Inner(r), start)
}

// SendIQElement is like SendIQ except that it wraps the payload in an
// Info/Query (IQ) element.
// For more information see SendIQ.
//
// SendIQElement is safe for concurrent use by multiple goroutines.
func (s *Session) SendIQElement(ctx context.Context, payload xml.TokenReader, iq stanza.IQ) (xmlstream.TokenReadCloser, error) {
	return s.SendIQ(ctx, iq.Wrap(payload))
}

// EncodeIQ is like Encode except that it returns an error if v does not marshal
// to an IQ stanza and like SendIQ it blocks until a response is received.
// For more information see SendIQ.
//
// EncodeIQ is safe for concurrent use by multiple goroutines.
func (s *Session) EncodeIQ(ctx context.Context, v interface{}) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(v)
	if err != nil {
		return nil, err
	}
	return s.SendIQ(ctx, r)
}

// EncodeIQElement is like EncodeIQ except that it wraps the payload in an
// Info/Query (IQ) element.
// For more information see SendIQ.
//
// EncodeIQElement is safe for concurrent use by multiple goroutines.
func (s *Session) EncodeIQElement(ctx context.Context, payload interface{}, iq stanza.IQ) (xmlstream.TokenReadCloser, error) {
	r, err := marshal.TokenReader(payload)
	if err != nil {
		return nil, err
	}
	return s.SendIQElement(ctx, r, iq)
}

// UnmarshalIQ is like SendIQ except that error replies are unmarshaled into a
// stanza.Error and returned and otherwise the response payload is unmarshaled
// into v.
// For more information see SendIQ.
//
// UnmarshalIQ is safe for concurrent use by multiple goroutines.
func (s *Session) UnmarshalIQ(ctx context.Context, iq xml.TokenReader, v interface{}) error {
	return unmarshalIQ(ctx, iq, v, s)
}

// UnmarshalIQElement is like UnmarshalIQ but it wraps a payload in the provided IQ.
// For more information see SendIQ.
//
// UnmarshalIQElement is safe for concurrent use by multiple goroutines.
func (s *Session) UnmarshalIQElement(ctx context.Context, payload xml.TokenReader, iq stanza.IQ, v interface{}) error {
	return unmarshalIQ(ctx, iq.Wrap(payload), v, s)
}

// IterIQ is like SendIQ except that error replies are unmarshaled into a
// stanza.Error and returned and otherwise an iterator over the children of the
// response payload is returned.
// For more information see SendIQ.
//
// IterIQ is safe for concurrent use by multiple goroutines.
func (s *Session) IterIQ(ctx context.Context, iq xml.TokenReader) (*xmlstream.Iter, *xml.StartElement, error) {
	return iterIQ(ctx, iq, s)
}

// IterIQElement is like IterIQ but it wraps a payload in the provided IQ.
// For more information see SendIQ.
//
// IterIQElement is safe for concurrent use by multiple goroutines.
func (s *Session) IterIQElement(ctx context.Context, payload xml.TokenReader, iq stanza.IQ) (*xmlstream.Iter, *xml.StartElement, error) {
	return iterIQ(ctx, iq.Wrap(payload), s)
}

func iterIQ(ctx context.Context, iq xml.TokenReader, s *Session) (_ *xmlstream.Iter, _ *xml.StartElement, e error) {
	resp, err := s.SendIQ(ctx, iq)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if e != nil {
			/* #nosec */
			resp.Close()
		}
	}()

	tok, err := resp.Token()
	if err != nil {
		return nil, nil, err
	}

	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil, nil, fmt.Errorf("stanza: expected IQ start token, got %T %[1]v", tok)
	}
	_, err = stanza.UnmarshalIQError(resp, start)
	if err != nil {
		return nil, nil, err
	}

	// Pop the payload start token, we want to iterate over its children.
	tok, err = resp.Token()
	start, _ = tok.(xml.StartElement)

	// Discard early EOF so that the iterator doesn't end up returning it.
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	return xmlstream.NewIter(resp), &start, nil
}

func unmarshalIQ(ctx context.Context, iq xml.TokenReader, v interface{}, s *Session) (e error) {
	resp, err := s.SendIQ(ctx, iq)
	if err != nil {
		return err
	}
	defer func() {
		ee := resp.Close()
		if e == nil {
			e = ee
		}
	}()

	tok, err := resp.Token()
	if err != nil {
		return err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return fmt.Errorf("stanza: expected IQ start token, got %T %[1]v", tok)
	}

	_, err = stanza.UnmarshalIQError(resp, start)
	if err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	payload := xmlstream.Inner(resp)
	d := xml.NewTokenDecoder(payload)
	startTok, err := d.Token()
	switch err {
	case io.EOF:
		return nil
	case nil:
	default:
		return err
	}
	start = startTok.(xml.StartElement)
	return d.DecodeElement(v, &start)
}
