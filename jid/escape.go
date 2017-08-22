// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"unicode/utf8"

	"github.com/mpvl/textutil"
)

var (
	// Escape is a transform that maps escapable runes to their escaped form as
	// defined in XEP-0106: JID Escaping.
	Escape = textutil.NewTransformerFromFunc(escape)

	// Unescape is a transform that maps valid escape sequences to their unescaped
	// form as defined in XEP-0106: JID Escaping.
	Unescape = textutil.NewTransformerFromFunc(unescape)
)

// EscapedChars is a string composed of all the characters that will be escaped
// or unescaped by the transformers in this package (in no particular order).
const EscapedChars = ` "&'/:<>@\`

func escape(s textutil.State) {
	switch r, _ := s.ReadRune(); r {
	case ' ':
		s.WriteString(`\20`)
	case '"':
		s.WriteString(`\22`)
	case '&':
		s.WriteString(`\26`)
	case '\'':
		s.WriteString(`\27`)
	case '/':
		s.WriteString(`\2f`)
	case ':':
		s.WriteString(`\3a`)
	case '<':
		s.WriteString(`\3c`)
	case '>':
		s.WriteString(`\3e`)
	case '@':
		s.WriteString(`\40`)
	case '\\':
		s.WriteString(`\5c`)
	default:
		s.WriteRune(r)
	}
}

// fmt.Printf("% x", EscapedChars):
// 20 22 26 27 2f 3a 3c 3e 40 5c
func unescape(s textutil.State) {
	if r, _ := s.ReadRune(); r != '\\' {
		s.WriteRune(r)
		return
	}

	// TODO: There's probably a better way to do this than generate a giant
	// switch/case tree.

	r, n := s.ReadRune()
	switch r {
	case utf8.RuneError:
		if n == 0 {
			s.WriteRune('\\')
			return
		}
		s.WriteRune(r)
		return
	case '2':
		switch r2, n := s.ReadRune(); r2 {
		case utf8.RuneError:
			if n == 0 {
				s.WriteRune('\\')
				s.WriteRune(r)
				return
			}
			s.WriteRune(r)
			s.WriteRune(r2)
			return
		case '0':
			s.WriteRune(' ')
		case '2':
			s.WriteRune('"')
		case '6':
			s.WriteRune('&')
		case '7':
			s.WriteRune('\'')
		case 'f':
			s.WriteRune('/')
		default:
			s.WriteRune('\\')
			s.WriteRune(r)
			s.WriteRune(r2)
		}
	case '3':
		switch r2, _ := s.ReadRune(); r2 {
		case utf8.RuneError:
			if n == 0 {
				s.WriteRune('\\')
				s.WriteRune(r)
				return
			}
			s.WriteRune(r)
			s.WriteRune(r2)
			return
		case 'a':
			s.WriteRune(':')
		case 'c':
			s.WriteRune('<')
		case 'e':
			s.WriteRune('>')
		default:
			s.WriteRune('\\')
			s.WriteRune(r)
			s.WriteRune(r2)
		}
	case '4':
		r2, n := s.ReadRune()
		switch r2 {
		case utf8.RuneError:
			if n == 0 {
				s.WriteRune('\\')
				s.WriteRune(r)
				return
			}
			s.WriteRune(r)
			s.WriteRune(r2)
		case '0':
			s.WriteRune('@')
		default:
			s.WriteRune('\\')
			s.WriteRune(r)
			s.WriteRune(r2)
		}
	case '5':
		r2, _ := s.ReadRune()
		switch r2 {
		case utf8.RuneError:
			if n == 0 {
				s.WriteRune('\\')
				s.WriteRune(r)
				return
			}
			s.WriteRune(r)
			s.WriteRune(r2)
		case 'c':
			s.WriteRune('\\')
		default:
			s.WriteRune('\\')
			s.WriteRune(r)
			s.WriteRune(r2)
		}
	default:
		s.WriteRune('\\')
		s.WriteRune(r)
	}
}
