// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package styling implements XEP-0393: Message Styling.
//
// For more information see:
// https://xmpp.org/extensions/xep-0393.html
//
// BE ADVISED: This package is experimental and the API is subject to change.
package styling

import (
	"bufio"
	"io"
)

// Style represents the currently active styles and blocks.
// For example, bytes between the styling directives in the span "_*Strong and
// emph*_" would have the style "Strong|Emph".
// Styling directives will have a marker indicating whether they are the start
// or end directive as well as the format itself.
// For example, the first "_" in the previous example would have the style:
// "StartEmph|Emph".
type Style uint32

// A list of possible styles and masks for accessing them.
const (
	// Spans
	Strong Style = 1 << iota
	Emph
	Pre
	Strike

	// Blocks
	PreBlock
	QuoteBlock

	// Masks
	Span  = Strong | Emph | Pre | Strike
	Block = PreBlock | QuoteBlock
)

// Parser reads message styling data from an underlying reader and returns the
// style of each byte.
type Parser struct {
	r *preBlockParser
}

// NewParser returns a Parser that reads data from the provided io.Reader.
// If r is not a bufio.Reader, Parser does its own buffering.
func NewParser(r io.Reader) Parser {
	p := Parser{}
	var buf *bufio.Reader
	if rr, ok := r.(*bufio.Reader); ok {
		buf = rr
	} else {
		buf = bufio.NewReader(r)
	}
	p.r = &preBlockParser{r: buf}
	return p
}

// Read reads data from the underlying reader and stops when the style would
// change.
func (p Parser) Read(b []byte) (n int, err error) {
	return p.r.Read(b)
}

// Style returns the style of the last byte read from the underlying reader.
func (p Parser) Style() Style {
	return p.r.Style()
}

type parser interface {
	io.Reader
	Style() Style
}
