// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"code.google.com/p/go.net/idna"
	"code.google.com/p/go.text/unicode/norm"

	"encoding/xml"
	"errors"
	"net"
	"strings"
	"unicode/utf8"
)

// Define some reusable error messages.
const (
	ERROR_INVALID_STRING    = "String is not valid UTF-8"
	ERROR_EMPTY_PART        = "JID parts must be > 0 bytes"
	ERROR_LONG_PART         = "JID parts must be < 1023 bytes"
	ERROR_LONG_DOMAIN_LABEL = "Domain names must be ≤ 63 chars per label"
	ERROR_LONG_DOMAIN_NAME  = "Domain names must be ≤ 253 chars"
	ERROR_LONG_DOMAIN_BYTES = "Domain names must be ≤ 255 octets"
	ERROR_INVALID_JID       = "String is not a valid JID"
	ERROR_ILLEGAL_RUNE      = "String contains an illegal chartacter"
	ERROR_ILLEGAL_SPACE     = "String contains illegal whitespace"
)

// NF is the Unicode normalization form to use. According to RFC 6122:
//
//      This profile specifies the use of Unicode Normalization Form KC, as
//      described in [STRINGPREP].
//
const NF norm.Form = norm.NFKC

// JID structs should not create one of these directly; instead, use the
// `NewJID()` function or the `jid.FromString(string)` method.
type JID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// NewJID creates a new JID from the given string.
func NewJID(s string) (JID, error) {
	j := JID{}
	err := j.FromString(s)
	return j, err
}

// Equals tests for JID equality by testing the three individual parts of a JID
// (localpart, domainpart, and resourcepart).
func (j *JID) Equals(jid2 JID) bool {
	domainpart, err := j.DomainPart()
	// Supressing an error, but if the domainpart errors it should never be equal.
	if err != nil {
		return false
	}
	domainpart2, err := jid2.DomainPart()
	if err != nil {
		return false
	}
	return (j.LocalPart() == jid2.LocalPart() && domainpart == domainpart2 && j.ResourcePart() == jid2.ResourcePart())
}

// LocalPart gets the localpart of a JID (eg "username").
func (j *JID) LocalPart() string {
	return j.localpart
}

// DomainPart gets the domainpart of a JID (eg. "example.net").
func (j *JID) DomainPart() (string, error) {
	return idna.ToUnicode(j.domainpart)
}

// ResourcePart gets the resourcepart of a JID (eg. "mobile").
func (j *JID) ResourcePart() string {
	return j.resourcepart
}

// NormalizeJIDPart verifies that the JID part is valid and returns a normalized
// string. You do NOT need to do this before passing parts to `NewJID()` or any
// of the `SetPart` methods; they handle validation and normalization for you.
// Eventually, this should be replaced with a proper stringprep implementation.
func NormalizeJIDPart(part string) (string, error) {
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
	// TODO: Is there no function or method to just do this?
	case len(strings.Fields("'"+normalized+"'")) != 1:
		// There should be no whitespace in the normalized part.
		return "", errors.New(ERROR_ILLEGAL_SPACE)
		// TODO: Use a proper stringprep library to make sure this is all correct.
	default:
		return normalized, nil
	}
}

// NormalizeResourcePart verifies that the JID resource part is valid and
// returns a normalized string. You probably do NOT need to call this manually,
// as creating a JID handles this for you. Eventually, this should be replaced
// with a proper stringprep implementation.
func NormalizeResourcePart(part string) (string, error) {
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
	// TODO: Is there no function or method to just do this?
	case len(strings.Fields("'"+normalized+"'")) != 1:
		// There should be no whitespace in the normalized part.
		return "", errors.New(ERROR_ILLEGAL_SPACE)
		// TODO: Use a proper stringprep library to make sure this is all correct.
	default:
		return normalized, nil
	}
}

// SetLocalPart sets the localpart of a JID and verifies that it is a
// valid/normalized UTF-8 string which is greater than 0 bytes and less than
// 1023 bytes.
func (j *JID) SetLocalPart(localpart string) error {
	normalized, err := NormalizeJIDPart(localpart)
	if err != nil {
		return err
	}
	(*j).localpart = normalized
	return nil
}

// SetDomainPart sets the domainpart of a JID and verify that it is a
// valid/normalized UTF-8 string which is greater than 0 bytes and less than
// 1023 bytes.
func (j *JID) SetDomainPart(domainpart string) error {

	// From RFC 6122 §2.2 Domainpart:
	//
	//     If the domainpart includes a final character considered to be a label
	//     separator (dot) by [IDNA2003] or [DNS], this character MUST be stripped
	//     from the domainpart before the JID of which it is a part is used for
	//     the purpose of routing an XML stanza, comparing against another JID, or
	//     constructing an [XMPP‑URI]. In particular, the character MUST be
	//     stripped before any other canonicalization steps are taken, such as
	//     application of the [NAMEPREP] profile of [STRINGPREP] or completion of
	//     the ToASCII operation as described in [IDNA2003].
	//
	domainpart = strings.TrimRight(domainpart, ".")

	normalized, err := idna.ToASCII(domainpart)
	if err != nil {
		return err
	}
	// Remove brackets if they already exist so that we can validate IPv6
	// TODO: Check if brackets exist and don't allow them if this isn't a v6 j
	normalized = strings.TrimPrefix(normalized, "[")
	normalized = strings.TrimSuffix(normalized, "]")
	// If the domain is a valid IPv6 j without brackets (it's a valid IP and
	// does not fit in 4 bytes), wrap it in brackets.
	// TODO: This is not very future proof.
	if ip := net.ParseIP(normalized); ip != nil && ip.To4() == nil {
		normalized = "[" + normalized + "]"
	}
	j.domainpart = normalized
	return nil
}

// SetResourcePart sets the resourcepart of a JID and verifies that it is a
// valid/normalized UTF-8 string which is greater than 0 bytes and less than
// 1023 bytes.
func (j *JID) SetResourcePart(resourcepart string) error {
	normalized, err := NormalizeResourcePart(resourcepart)
	if err != nil {
		return err
	}
	j.resourcepart = normalized
	return nil
}

// String converts the full JID to a string.
func (j *JID) String() string {
	out, _ := j.DomainPart()
	if lp := j.LocalPart(); lp != "" {
		out = j.LocalPart() + "@" + out
	}
	if rp := j.ResourcePart(); rp != "" {
		out = out + "/" + rp
	}
	return out
}

// Bare returns the bare JID (no resourcepart) as a string.
func (j *JID) Bare() (string, error) {
	out, err := j.DomainPart()
	if lp := j.LocalPart(); lp != "" {
		out = lp + "@" + out
	}
	return out, err
}

// FromString sets the fields in an existing JID from a string.
func (j *JID) FromString(s string) error {

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
	// So don't normalize until after we've checked the various parts.

	// Trim any whitespace before we begin.
	s = strings.TrimSpace(s)

	// Do not allow whitespace elsewhere in the string…
	if len(strings.Fields(s)) != 1 {
		return errors.New(ERROR_ILLEGAL_SPACE)
	}

	atCount := strings.Count(s, "@")
	slashCount := strings.Count(s, "/")
	atLoc := strings.IndexRune(s, '@')
	slashLoc := strings.IndexRune(s, '/')

	switch {
	case atCount == 0 && slashCount == 0:
		// domainpart only (eg. "example.net" or "example")
		err := j.SetDomainPart(s)
		if err != nil {
			return err
		}

	case atCount == 1 && slashCount == 0:
		// Bare JID ("test@example.net" or "test@example")
		if atLoc == 0 || atLoc == len(s)-1 {
			return errors.New(ERROR_EMPTY_PART)
		}
		err := j.SetLocalPart(s[0:atLoc])
		if err != nil {
			return err
		}
		err = j.SetDomainPart(s[atLoc+1:])
		if err != nil {
			return err
		}

	case slashCount > 0 && (atCount == 0 || atLoc > slashLoc):
		// domainpart + resourcepart (eg. "example/rp" or "example/@/")
		if slashLoc == 0 || slashLoc == len(s)-1 {
			// Error if JID is of the form "/jid" or "jid/" ("jid//" is okay)
			return errors.New(ERROR_EMPTY_PART)
		}
		err := j.SetDomainPart(s[0:slashLoc])
		if err != nil {
			return err
		}
		err = j.SetResourcePart(s[slashLoc+1:])
		if err != nil {
			return err
		}

	case slashCount > 0 && atCount > 0 && atLoc < slashLoc:
		// Full JID (eg. "test@example.net/resourcepart" or "test@example.net/@/")
		last := len(s) - 1
		if atLoc == 0 || slashLoc == 0 || atLoc == last || slashLoc == last || slashLoc == atLoc+1 {
			return errors.New(ERROR_EMPTY_PART)
		}
		err := j.SetLocalPart(s[0:atLoc])
		if err != nil {
			return err
		}
		err = j.SetDomainPart(s[atLoc+1 : slashLoc])
		if err != nil {
			return err
		}
		err = j.SetResourcePart(s[slashLoc+1:])
		if err != nil {
			return err
		}

	default: // Too many '@' or '/' symbols
		return errors.New(ERROR_ILLEGAL_RUNE)
	}

	return nil
}

// MarshalXMLAttr marshals the JID as an XML attriute for use with the
// `encoding/xml' package.
func (j *JID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}
