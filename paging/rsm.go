// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature

// Package paging implements result set management.
package paging // import "mellium.im/xmpp/paging"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

// Namespaces used by this package.
const (
	NS = "http://jabber.org/protocol/rsm"
)

// Iter provides a mechanism for iterating over the children of an XML element.
// Successive calls to Next will step through each child, returning its start
// element and a reader that is limited to the remainder of the child.
//
// If the results indicate that there is another page of data, the paging child
// is skipped and the various paging methods will return queries that can be
// used to fetch the next and/or previous pages.
type Iter struct {
	iter        *xmlstream.Iter
	nextPageSet *RequestNext
	prevPageSet *RequestPrev
	curSet      *Set
	err         error
	max         uint64
}

// NewIter returns a new iterator that iterates over the children of the most
// recent start element already consumed from r.
func NewIter(r xml.TokenReader, max uint64) *Iter {
	return WrapIter(xmlstream.NewIter(r), max)
}

// WrapIter returns a new iterator that supports paging from an existing
// xmlstream.Iter.
func WrapIter(iter *xmlstream.Iter, max uint64) *Iter {
	return &Iter{
		iter: iter,
		max:  max,
	}
}

// Close indicates that we are finished with the given iterator. Calling it
// multiple times has no effect.
//
// If the underlying TokenReader is also an io.Closer, Close calls the readers
// Close method.
func (i *Iter) Close() error {
	return i.iter.Close()
}

// Current returns a reader over the most recent child.
func (i *Iter) Current() (*xml.StartElement, xml.TokenReader) {
	return i.iter.Current()
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}
	return i.iter.Err()
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil {
		return false
	}
	hasNext := i.iter.Next()
	if hasNext {
		start, r := i.iter.Current()
		if start != nil && start.Name.Local == "set" && start.Name.Space == NS {
			i.nextPageSet = nil
			i.prevPageSet = nil
			i.curSet = &Set{}
			i.err = xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), r)).Decode(i.curSet)
			if i.err != nil {
				return false
			}
			if i.curSet.First.ID != "" {
				i.prevPageSet = &RequestPrev{
					Before: i.curSet.First.ID,
					Max:    i.max,
				}
			}
			if i.curSet.Last != "" {
				i.nextPageSet = &RequestNext{
					After: i.curSet.Last,
					Max:   i.max,
				}
			}
			return i.Next()
		}
	}
	return hasNext
}

// NextPage returns a value that can be used to construct a new iterator that
// queries for the next page.
//
// It is only guaranteed to be set once iteration is finished, or when the
// iterator is closed without error and may be nil.
func (i *Iter) NextPage() *RequestNext {
	return i.nextPageSet
}

// PreviousPage returns a value that can be used to construct a new iterator that
// queries for the previous page.
//
// It is only guaranteed to be set once iteration is finished, or when the
// iterator is closed without error and may be nil.
func (i *Iter) PreviousPage() *RequestPrev {
	return i.prevPageSet
}

// CurrentPage returns information about the current page.
//
// It is only guaranteed to be set once iteration is finished, or when the
// iterator is closed without error and may be nil.
func (i *Iter) CurrentPage() *Set {
	return i.curSet
}
