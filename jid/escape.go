// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid

import (
	"bytes"

	"golang.org/x/text/transform"
)

// Transformer implements the transform.Transformer and
// transform.SpanningTransformer interfaces.
//
// For more information see golang.org/x/text/transform or the predefined Escape
// and Unescape transformers.
type Transformer struct {
	t transform.SpanningTransformer
}

// Reset implements the transform.Transformer interface.
func (t Transformer) Reset() { t.t.Reset() }

// Transform implements the transform.Transformer interface.
func (t Transformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	return t.t.Transform(dst, src, atEOF)
}

// Span implements the transform.SpanningTransformer interface.
func (t Transformer) Span(src []byte, atEOF bool) (n int, err error) {
	return t.t.Span(src, atEOF)
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

var (
	// Escape is a transform that maps escapable runes to their escaped form as
	// defined in XEP-0106: JID Escaping.
	Escape Transformer = Transformer{escapeMapping{}}

	// Unescape is a transform that maps valid escape sequences to their unescaped
	// form as defined in XEP-0106: JID Escaping.
	Unescape Transformer = Transformer{unescapeMapping{}}
)

const escape = ` "&'/:<>@\`

type escapeMapping struct {
	transform.NopResetter
}

func (escapeMapping) Span(src []byte, atEOF bool) (n int, err error) {
	switch idx := bytes.IndexAny(src, escape); idx {
	case -1:
		return len(src), nil
	default:
		return idx, transform.ErrEndOfSpan
	}
}

func (t escapeMapping) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		idx := bytes.IndexAny(src[nSrc:], escape)
		switch idx {
		case -1:
			n := copy(dst[nDst:], src[nSrc:])
			nDst += n
			nSrc += n
			if nSrc < len(src) {
				return nDst, nSrc, transform.ErrShortDst
			}
		default:
			n := copy(dst[nDst:], src[nSrc:nSrc+idx])
			nDst += n
			nSrc += n
			if n != idx-nSrc {
				return nDst, nSrc, transform.ErrShortDst
			}
			c := src[nSrc]
			n = copy(dst[nDst:], []byte{
				'\\',
				"0123456789abcdef"[c>>4],
				"0123456789abcdef"[c&15],
			})
			nDst += n
			nSrc++
			if n != 3 {
				return nDst, nSrc, transform.ErrShortDst
			}
		}
	}
	return
}

type unescapeMapping struct {
	transform.NopResetter
}

// TODO: Be more specific. Only check if it's the starting character in any
//       valid escape sequence.

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// I just wrote these all out because it's a lot faster and not likely to
// change; is it really worth the confusing logic though?
func shouldUnescape(s []byte) bool {
	return (s[0] == '2' && (s[1] == '0' || s[1] == '2' || s[1] == '6' || s[1] == '7' || s[1] == 'f' || s[1] == 'F')) || (s[0] == '3' && (s[1] == 'a' || s[1] == 'A' || s[1] == 'c' || s[1] == 'C' || s[1] == 'e' || s[1] == 'E')) || (s[0] == '4' && s[1] == '0') || (s[0] == '5' && (s[1] == 'c' || s[1] == 'C'))
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func (unescapeMapping) Span(src []byte, atEOF bool) (n int, err error) {
	for n < len(src) {
		if src[n] != '\\' {
			n++
			continue
		}

		switch n {
		case len(src) - 1:
			// The last character is the escape char.
			if atEOF {
				return len(src), nil
			}
			return n, transform.ErrShortSrc
		case len(src) - 2:
			if atEOF || !ishex(src[n+1]) {
				return len(src), nil
			}
			return n, transform.ErrShortSrc
		}

		if shouldUnescape(src[n+1 : n+3]) {
			// unhex(s[n+1])<<4 | unhex(s[n+2])
			return n, transform.ErrEndOfSpan
		}
		n++
	}
	return
}

func (t unescapeMapping) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		idx := bytes.IndexRune(src[nSrc:], '\\')

		switch {
		case idx == -1 || (idx == len(src[nSrc:])-1 && atEOF):
			// No unescape sequence exists, or the escape sequence is at the end but
			// there aren't enough following characters to make it valid, so copy to
			// the end.
			n := copy(dst[nDst:], src[nSrc:])
			nDst += n
			nSrc += n
			if nSrc < len(src) {
				return nDst, nSrc, transform.ErrShortDst
			}
			return
		case idx == len(src[nSrc:])-1:
			// The last character is the escape char and this isn't the EOF
			n := copy(dst[nDst:], src[nSrc:nSrc+idx])
			nDst += n
			nSrc += n
			if n != idx {
				return nDst, nSrc, transform.ErrShortDst
			}
			return nDst, nSrc, transform.ErrShortSrc
		case idx == len(src[nSrc:])-2:
			if atEOF || !ishex(src[nSrc+idx+1]) {
				n := copy(dst[nDst:], src[nSrc:])
				nDst += n
				nSrc += n
				if nSrc < len(src) {
					return nDst, nSrc, transform.ErrShortDst
				}
				return
			}
			n := copy(dst[nDst:], src[nSrc:nSrc+idx])
			nDst += n
			nSrc += n
			if n != idx {
				return nDst, nSrc, transform.ErrShortDst
			}
			return nDst, nSrc, transform.ErrShortSrc
		}

		if shouldUnescape(src[nSrc+idx+1 : nSrc+idx+3]) {
			n := copy(dst[nDst:], src[nSrc:nSrc+idx])
			nDst += n
			nSrc += n
			if n != idx {
				return nDst, nSrc, transform.ErrShortDst
			}
			if n == 0 {
				n++
			}
			n = copy(dst[nDst:], []byte{
				unhex(src[nSrc+n])<<4 | unhex(src[nSrc+n+1]),
			})
			nDst += n
			nSrc += 3
			if n != 1 {
				return nDst, nSrc, transform.ErrShortDst
			}
			continue
		}
		n := copy(dst[nDst:], src[nSrc:nSrc+idx+1])
		nDst += n
		nSrc += n
		if n != idx+1 {
			return nDst, nSrc, transform.ErrShortDst
		}
	}
	return
}
