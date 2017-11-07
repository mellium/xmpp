// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package styling implements the simple text formatting language from the
// Message Styling proto-XEP.
//
// Currently it only supports preformatted text.
//
// For more information see:
// https://xmpp.org/extensions/inbox/styling.html
//
// BE ADVISED: This package is experimental and the API is subject to change.
package styling

import (
	"bytes"

	"golang.org/x/text/transform"
)

// HTML transforms the styled text into HTML, keeping the styling characters.
// It does not perform escaping of existing HTML entities.
func HTML() Transformer {
	return New(
		Bold("<strong>", "</strong>"),
		Italic("<it>", "</it>"),
		Mono("<tt>", "</tt>"),
		Pre("<pre>", "</pre>"),
		Quote("<quote>", "</quote>"),
		Strike("<s>", "</s>"),
	)
}

// HTMLNoStyle transforms the styled text into HTML, hiding the styling
// characters.
// It does not perform escaping of existing HTML entities.
func HTMLNoStyle() Transformer {
	return New(
		Bold("<strong>", "</strong>"),
		Italic("<it>", "</it>"),
		Mono("<tt>", "</tt>"),
		Pre("<pre>", "</pre>"),
		Quote("<quote>", "</quote>"),
		Strike("<s>", "</s>"),
		NoStyle(),
	)
}

// Markdown transforms the styled text into Markdown.
func Markdown() Transformer {
	return New(
		Bold("**", "**"),
		Italic("_", "_"),
		Mono("`", "`"),
		Pre("```\n", "```"),
		Quote(">", ""),
		Strike("~", "~"),
		NoStyle(),
	)
}

// Transformer is a stateless transformer that converts text from the format
// defined in the styling ProtoXEP to to a different representation such as HTML
// or LaTeX.
type Transformer struct {
	transform.NopResetter

	boldEnd     []byte
	boldStart   []byte
	italicEnd   []byte
	italicStart []byte
	monoEnd     []byte
	monoStart   []byte
	preEnd      []byte
	preStart    []byte
	quoteEnd    []byte
	quoteStart  []byte
	strikeEnd   []byte
	strikeStart []byte
	noStyle     bool
}

// Bytes returns a new byte slice with the result of applying t to b.
func (t Transformer) Bytes(b []byte) []byte {
	b, _, _ = transform.Bytes(t, b)
	return b
}

// String returns a string with the result of applying t to s.
func (t Transformer) String(s string) string {
	s, _, _ = transform.String(t, s)
	return s
}

// Transform implements the transform.Transformer interface for Transformer.
func (t Transformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	const (
		startPre = "```\n"
		endPre   = "```"
	)

	// If the src buffer starts with "```" expand until the ending "```" and
	// copy as a preformatted text block.
	if bytes.HasPrefix(src, []byte(startPre)) {
		// Scan until the end of the pre or EOF
		var toEnd bool
		endLoc := bytes.Index(src[len(startPre):], []byte(endPre))
		switch {
		case !atEOF && endLoc == -1:
			return 0, 0, transform.ErrShortSrc
		case atEOF && endLoc == -1:
			toEnd = true
			endLoc = len(src)
		case endLoc != -1:
			endLoc += len(startPre) + len(endPre)
		}

		start := 0
		if t.noStyle {
			start = len(startPre)
			if !toEnd {
				endLoc -= len(endPre)
			}
		}
		if len(dst) < (endLoc-start)+len(t.preStart)+len(t.preEnd) {
			return 0, 0, transform.ErrShortDst
		}
		nDst += copy(dst, t.preStart)
		n := copy(dst[nDst:], src[start:endLoc])
		nDst += n
		nSrc += n
		if t.noStyle {
			nSrc += len(startPre)
			if !toEnd {
				nSrc += len(endPre)
			}
		}
		nDst += copy(dst[nDst:], t.preEnd)
		return nDst, nSrc, nil
	}

	// Buffer at least one line.
	line := bytes.IndexByte(src, '\n')
	switch {
	case !atEOF && line == -1:
		return 0, 0, transform.ErrShortSrc
	case atEOF && line == -1:
		line = len(src) - 1
	}
	if len(dst) < line+1 {
		return 0, 0, transform.ErrShortDst
	}
	n := copy(dst, src[:line+1])
	nDst += n
	nSrc += n
	return nDst, nSrc, nil
}

// New returns a new Transformer that converts text using the provided options.
func New(o ...Option) Transformer {
	t := Transformer{}
	for _, opt := range o {
		opt(&t)
	}
	return t
}

// Option configures how a Transformer is applied to the text.
type Option func(*Transformer)

// NoStyle hides the styling directives in the output text.
func NoStyle() Option {
	return func(t *Transformer) {
		t.noStyle = true
	}
}

// Bold configures what the start and end "*" or "**" sequences should become.
func Bold(start, end string) Option {
	return func(t *Transformer) {
		t.boldEnd = []byte(end)
		t.boldStart = []byte(start)
	}
}

// Italic configures what the start and end "_" or "__" sequences should become.
func Italic(start, end string) Option {
	return func(t *Transformer) {
		t.italicEnd = []byte(end)
		t.italicStart = []byte(start)
	}
}

// Mono configures what the start and end "`" or "``" sequences should become.
func Mono(start, end string) Option {
	return func(t *Transformer) {
		t.monoEnd = []byte(end)
		t.monoStart = []byte(start)
	}
}

// Pre configures what the start and end "```" sequences should become.
func Pre(start, end string) Option {
	return func(t *Transformer) {
		t.preEnd = []byte(end)
		t.preStart = []byte(start)
	}
}

// Quote configures what should be inserted at the beginning and end of a
// quotation block.
func Quote(start, end string) Option {
	return func(t *Transformer) {
		t.quoteEnd = []byte(end)
		t.quoteStart = []byte(start)
	}
}

// Strike configures what the start and end "~" or "~~" sequences should become.
func Strike(start, end string) Option {
	return func(t *Transformer) {
		t.strikeEnd = []byte(end)
		t.strikeStart = []byte(start)
	}
}
