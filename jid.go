// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"code.google.com/p/go.text/unicode/norm"
	// TODO: Use a proper stringprep library like "code.google.com/p/go-idn/idna"
	"errors"
	"net"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Define some reusable error messages.
const (
	ERROR_INVALID_STRING = "String is not valid UTF-8"
	ERROR_EMPTY_PART     = "JID parts must be greater than 0 bytes"
	ERROR_LONG_PART      = "JID parts must be less than 1023 bytes"
	ERROR_NO_RESOURCE    = "String is a bare JID"
	ERROR_INVALID_JID    = "String is not a valid JID"
	ERROR_ILLEGAL_RUNE   = "String contains an illegal chartacter"
	ERROR_ILLEGAL_SPACE  = "String contains illegal whitespace"
)

// The Unicode normalization form to use. According to RFC 6122:
//
//      This profile specifies the use of Unicode Normalization Form KC, as
//      described in [STRINGPREP].
//
const NF norm.Form = norm.NFKC

// The jid struct is left unexported so that setters (which provide validation)
// must be used when creating or modifying JIDs.
type jid struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// Create a new JID from the given string.
func NewJID(s string) (jid, error) {
	j := jid{}
	err := j.FromString(s)
	return j, err
}

// Get the local part of a JID
func (address *jid) LocalPart() string {
	return address.localpart
}

// Get the domainpart of a JID
func (address *jid) DomainPart() string {
	return address.domainpart
}

// Get the resourcepart of a JID
func (address *jid) ResourcePart() string {
	return address.resourcepart
}

// Verify that the JID part is valid and return a normalized string.
func normalizeJIDPart(part string) (string, error) {
	switch normalized := NF.String(part); {
	case len(normalized) == 0:
		// The normalized length should be > 0 bytes
		return "", errors.New(ERROR_EMPTY_PART)
	case len(normalized) > 1023:
		// The normalized length should be ≤ 1023 bytes
		return "", errors.New(ERROR_LONG_PART)
	case !utf8.ValidString(part):
		// The original string should be valid UTF-8
		return "", errors.New(ERROR_INVALID_STRING)
	case strings.ContainsAny(part, "\"&'/:<>@"):
		// The original string should not contain any illegal characters. After
		// normalization some of these characters maybe present.
		return "", errors.New(ERROR_ILLEGAL_RUNE)
	case len(strings.Fields(normalized)) != 1:
		// There should be no whitespace in the normalized part.
		return "", errors.New(ERROR_ILLEGAL_SPACE)
		// TODO: Use a proper stringprep library to make sure this is all correct.
	default:
		return normalized, nil
	}
}

// Set the localpart of a JID and verify that it is a valid/normalized UTF-8
// string which is greater than 0 bytes and less than 1023 bytes.
func (address *jid) SetLocalPart(localpart string) error {
	normalized, err := normalizeJIDPart(localpart)
	if err != nil {
		return err
	}
	(*address).localpart = normalized
	return nil
}

// Set the domainpart of a JID and verify that it is a valid/normalized  UTF-8
// string which is greater than 0 bytes and less than 1023 bytes.
func (address *jid) SetDomainPart(domainpart string) error {
	normalized, err := normalizeJIDPart(domainpart)
	if err != nil {
		return err
	}
	// Remove brackets if they already exist so that we can validate IPv6
	// TODO: Check if brackets exist and don't allow them if this isn't a v6 address
	normalized = strings.TrimPrefix(normalized, "[")
	normalized = strings.TrimSuffix(normalized, "]")
	// If the domain is a valid IPv6 address without brackets (it's a valid IP and
	// does not fit in 4 bytes), wrap it in brackets.
	// TODO: This is not very future proof.
	if ip := net.ParseIP(normalized); ip != nil && ip.To4() == nil {
		normalized = "[" + normalized + "]"
	}
	// According to RFC 6122:
	// If the domainpart includes a final character considered to be a label
	// separator (dot) by [IDNA2003] or [DNS], this character MUST be stripped
	// from the domainpart before the JID of which it is a part is used for the
	// purpose of routing an XML stanza, comparing against another JID, or
	// constructing an [XMPP-URI].
	normalized = strings.TrimSuffix(normalized, ".")
	address.domainpart = normalized
	return nil
}

// Set the resourcepart of a JID and verify that it is a valid/normalized UTF-8
// string which is greater than 0 bytes and less than 1023 bytes.
func (address *jid) SetResourcePart(resourcepart string) error {
	normalized, err := normalizeJIDPart(resourcepart)
	if err != nil {
		return err
	}
	address.resourcepart = normalized
	return nil
}

// Return the full JID as a string
func (address *jid) String() string {
	return address.LocalPart() + "@" + address.DomainPart() + "/" + address.ResourcePart()
}

// Return the bare JID as a string
func (address jid) Bare() string {
	return address.LocalPart() + "@" + address.DomainPart()
}

// Set the JIDs properties from a string.
// Technically the only required part of a JID is the domainpart.
const JIDMatch = "[^@/]+@[^@/]+/[^@/]+"

func (address *jid) FromString(s string) error {
	// Make sure the string is valid UTF-8
	if !utf8.ValidString(s) {
		return errors.New(ERROR_INVALID_STRING)
	}
	// According to RFC 6122:
	//
	//     Implementation Note: When dividing a JID into its component parts, an
	//     implementation needs to match the separator characters '@' and '/'
	//     before applying any transformation algorithms, which might decompose
	//     certain Unicode code points to the separator characters (e.g., U+FE6B
	//     SMALL COMMERCIAL AT might decompose into U+0040 COMMERCIAL AT).
	//
	// So don't normalize before we check the regex.
	switch matched, err := regexp.MatchString(JIDMatch, s); {
	case err != nil:
		return err
	case !matched && !strings.ContainsRune(s, '/'):
		return errors.New(ERROR_NO_RESOURCE)
	case !matched:
		return errors.New(ERROR_INVALID_JID)
	}
	s = strings.TrimSpace(s)
	// Set the various parts of the JID
	atLoc := strings.IndexRune(s, '@')
	slashLoc := strings.IndexRune(s, '/')

	err := address.SetLocalPart(s[0:atLoc])
	if err != nil {
		return err
	}
	err = address.SetDomainPart(s[atLoc+1 : slashLoc])
	if err != nil {
		return err
	}
	err = address.SetResourcePart(s[slashLoc+1:])
	if err != nil {
		return err
	}
	return nil
}
