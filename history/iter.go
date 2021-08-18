// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history

import (
	"encoding/xml"
)

// Iter is an iterator over message history.
type Iter struct {
	err  error
	msgC chan xml.TokenReader
	cur  xml.TokenReader
	h    *Handler
	id   string
	res  Result
}

// Next advances the iterator
func (i *Iter) Next() bool {
	var ok bool
	i.cur, ok = <-i.msgC
	return ok
}

// Current returns the current message stream read from the iterator.
func (i *Iter) Current() xml.TokenReader {
	return i.cur
}

// Err returns any error encountered by the iterator.
func (i *Iter) Err() error {
	return i.err
}

// Result contains the results of the query after iteration has completed if no
// error was encountered.
func (i *Iter) Result() Result {
	return i.res
}

// Close stops iterating over this query.
// Future messages will still be received but will be handled by the fallback
// handler instead.
func (i *Iter) Close() error {
	i.h.remove(i.id)
	return nil
}
