// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package uri parses XMPP URI and IRI's as defined in RFC 5122.
//
// It also provides easy access to query components defined in XEP-0147: XMPP
// URI Scheme Query Components and the XMPP URI/IRI Querytypes registry.
package uri // import "mellium.im/xmpp/uri"

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"mellium.im/xmpp/jid"
)

var (
	errBadScheme = errors.New("uri: expected scheme xmpp")
)

// URI is a parsed XMPP URI or IRI.
type URI struct {
	*url.URL

	// ToAddr is the recipient address.
	ToAddr jid.JID

	// AuthAddr is empty if we should perform an action as the currently
	// authenticated account or ask the user to input the account to use.
	// Otherwise it is the auth address if present in an xmpp:// URI or IRI.
	AuthAddr jid.JID

	// Action is the first query component without a value and normally determines
	// the action to take when handling the URI. For example, the query string
	// might be ?join to join a chatroom, or ?message to send a message.
	//
	// For more information see XEP-0147: XMPP URI Scheme Query Components.
	Action string
}

// TODO: encoding and escaping, see
// https://tools.ietf.org/html/rfc5122#section-2.7.2

// Parse parses rawuri into a URI structure.
func Parse(rawuri string) (*URI, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "xmpp" {
		return nil, errBadScheme
	}

	uri := &URI{
		URL: u,
	}

	if u.Host != "" {
		// If an authentication address was provided (ie. the URI started with
		// `xmpp://'), parse it out and take the recipient address from the path.

		uri.AuthAddr, err = jid.New(u.User.Username(), u.Hostname(), "")
		if err != nil {
			return nil, err
		}
		if u.Path != "" {
			// Strip the root / and use the path as the JID.
			iri, err := toIRI(u.Path[1:], false)
			if err != nil {
				return nil, err
			}
			uri.ToAddr, err = jid.Parse(iri)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// If no auth address was provided (ie. the URI started with `xmpp:') take
		// the recipient address from the opaque part and ignore the user info.
		iri, err := toIRI(u.Opaque, true)
		if err != nil {
			return nil, err
		}
		uri.ToAddr, err = jid.Parse(iri)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range u.Query() {
		if len(v) == 0 || len(v) == 1 && v[0] == "" {
			uri.Action = k
			break
		}
	}

	return uri, err
}

// String reassembles the URI or IRI Into a valid IRI string.
func (u *URI) String() string {
	iri, _ := toIRI(u.URL.String(), true)
	return iri
}

// toIRI converts the URI to a valid IRI using the algorithm defined in RFC 3987
// §3.2.
// It does not validate that the input is a valid URI.
func toIRI(u string, needsUnescape bool) (string, error) {
	// 1.  Represent the URI as a sequence of octets in US-ASCII.
	//
	// 2.  Convert all percent-encodings ("%" followed by two hexadecimal
	//     digits) to the corresponding octets, except those corresponding
	//     to "%", characters in "reserved", and characters in US-ASCII not
	//     allowed in URIs.
	// TODO: using PathUnescape to create a new string is very inefficient, but
	// it's the only method available in the standard library for this.
	// In the future we should write an escape/unescaper that implements
	// "golang.org/x/text/transform".Transformer or simply appends to a buffer or
	// byte slice so that the next step can also be done in the same iteration
	// without creating yet another builder.
	var err error
	if needsUnescape {
		u, err = url.PathUnescape(u)
		if err != nil {
			return "", err
		}
	}

	// 3. Re-percent-encode any octet produced in step 2 that is not part
	//    of a strictly legal UTF-8 octet sequence.
	// 4. Re-percent-encode all octets produced in step 3 that in UTF-8
	//    represent characters that are not appropriate according to
	//    sections 2.2, 4.1, and 6.1.
	// 5. Interpret the resulting octet sequence as a sequence of characters
	//    encoded in UTF-8.
	u = escapeInvalidUTF8(u)

	return u, nil
}

// escapeInvalidUTF8 is like strings.ToValidUTF8 except that it replaces invalid
// UTF8 with % encoded versions of the invalid bytes instead of a fixed string.
func escapeInvalidUTF8(s string) string {
	// This function is a modified form of code copied from
	// go/src/strings/strings.go under the terms of Go's BSD license.
	// See the file LICENSE-GO for details.
	var b strings.Builder

	for i, c := range s {
		if !runeDisallowed(c, 1) {
			continue
		}

		r, wid := utf8.DecodeRuneInString(s[i:])
		if runeDisallowed(r, wid) {
			// 3 bytes in %AB.
			b.Grow(len(s) + 3*wid)
			_, err := b.WriteString(s[:i])
			if err != nil {
				panic(fmt.Errorf("error writing string to buffer: %w", err))
			}
			s = s[i:]
			break
		}
	}

	// Fast path for unchanged input
	if b.Cap() == 0 { // didn't call b.Grow above
		return s
	}

	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			i++
			err := b.WriteByte(c)
			if err != nil {
				panic(fmt.Errorf("error writing byte to buffer: %w", err))
			}
			continue
		}
		r, wid := utf8.DecodeRuneInString(s[i:])
		if runeDisallowed(r, wid) {
			for j := 0; j < wid; j++ {
				fmt.Fprintf(&b, "%%%0X", s[i+j:i+j+1])
			}
			i += wid
			continue
		}
		_, err := b.WriteString(s[i : i+wid])
		if err != nil {
			panic(fmt.Errorf("error writing remaining string to buffer: %w", err))
		}
		i += wid
	}

	return b.String()
}

func runeDisallowed(r rune, wid int) bool {
	switch r {
	case utf8.RuneError:
		// the various utf8.Decode methods return wid==1 on invalid rune. 0 means
		// empty string, other values won't be returned.
		return wid == 1
	case '\u200e', '\u200f', '\u202a', '\u202b', '\u202d', '\u202e', '\u202c':
		// RFC 3987 §4.1:
		//
		//     IRIs MUST NOT contain bidirectional formatting characters (LRM, RLM,
		//     LRE, RLE, LRO, RLO, and PDF).
		return true
	}
	return false
}
