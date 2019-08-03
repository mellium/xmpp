// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package iter provides a streaming iterator over an XML elements children.
//
// This will likely be moved to mellium.im/xmlstream once the API is finalized.
package iter

import (
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
)

// Iter provides a mechanism for streaming the children of an XML element.
// Successive calls to the Next method will step through each child, returning
// its start element and a reader that is limited to the remainder of the child.
type Iter struct {
	r       xml.TokenReader
	err     error
	next    *xml.StartElement
	cur     xml.TokenReader
	closed  bool
	discard xmlstream.TokenWriter
}

// New returns a new iterator that iterates over the children of the most recent start
// element already consumed from r.
func New(r xml.TokenReader) *Iter {
	iter := &Iter{
		r:       xmlstream.Inner(r),
		discard: xmlstream.Discard(),
	}
	return iter
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || i.closed {
		return false
	}

	// Consume the previous element before moving on to the next.
	if i.cur != nil {
		_, i.err = xmlstream.Copy(i.discard, i.cur)
		if i.err != nil {
			return false
		}
	}

	i.next = nil
	t, err := i.r.Token()
	if err != nil {
		if err != io.EOF {
			i.err = err
		}
		return false
	}

	if start, ok := t.(xml.StartElement); ok {
		i.next = &start
		i.cur = xmlstream.MultiReader(xmlstream.Inner(i.r), xmlstream.Token(i.next.End()))
		return true
	}
	return false
}

// Current returns a reader over the most recent child.
func (i *Iter) Current() (*xml.StartElement, xml.TokenReader) {
	return i.next, i.cur
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	return i.err
}

// Close indicates that we are finished with the given iterator.
// Calling it multiple times has no effect.
//
// If the underlying TokenReader is also an io.Closer, Close calls the readers
// Close method.
func (i *Iter) Close() error {
	if i.closed {
		return nil
	}

	i.closed = true
	_, err := xmlstream.Copy(i.discard, i.r)
	if err != nil {
		return err
	}
	if c, ok := i.r.(xmlstream.TokenReadCloser); ok {
		return c.Close()
	}
	return nil
}
