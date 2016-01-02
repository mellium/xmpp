// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"strings"
)

// JID defines methods that are common to all XMPP addresses (historically,
// "Jabber ID") implementations.
type JID interface {
	Localpart() string
	Domainpart() string
	Resourcepart() string
	Bare() JID

	Equal(other JID) bool

	fmt.Stringer
	xml.MarshalerAttr
	xml.UnmarshalerAttr
}

// SplitString splits out the localpart, domainpart, and resourcepart from a
// string representation of a JID. The parts are not guaranteed to be valid, and
// each part must be 1023 bytes or less.
func SplitString(s string) (localpart, domainpart, resourcepart string, err error) {

	// RFC 7622 §3.1.  Fundamentals:
	//
	//    Implementation Note: When dividing a JID into its component parts,
	//    an implementation needs to match the separator characters '@' and
	//    '/' before applying any transformation algorithms, which might
	//    decompose certain Unicode code points to the separator characters.
	//
	// so let's do that now. First we'll parse the domainpart using the rules
	// defined in §3.2:
	//
	//    The domainpart of a JID is the portion that remains once the
	//    following parsing steps are taken:
	//
	//    1.  Remove any portion from the first '/' character to the end of the
	//        string (if there is a '/' character present).
	parts := strings.SplitAfterN(
		s, "/", 2,
	)

	// If the resource part exists, make sure it isn't empty.
	if strings.HasSuffix(parts[0], "/") {
		if len(parts) == 2 && parts[1] != "" {
			resourcepart = parts[1]
		} else {
			err = errors.New("The resourcepart must be larger than 0 bytes")
			return
		}
	} else {
		resourcepart = ""
	}

	norp := strings.TrimSuffix(parts[0], "/")

	//    2.  Remove any portion from the beginning of the string to the first
	//        '@' character (if there is an '@' character present).

	nolp := strings.SplitAfterN(norp, "@", 2)

	if nolp[0] == "@" {
		err = errors.New("The localpart must be larger than 0 bytes")
		return
	}

	switch len(nolp) {
	case 1:
		domainpart = nolp[0]
		localpart = ""
	case 2:
		domainpart = nolp[1]
		localpart = strings.TrimSuffix(nolp[0], "@")
	}

	// We'll throw out any trailing dots on domainparts, since they're ignored:
	//
	//    If the domainpart includes a final character considered to be a label
	//    separator (dot) by [RFC1034], this character MUST be stripped from
	//    the domainpart before the JID of which it is a part is used for the
	//    purpose of routing an XML stanza, comparing against another JID, or
	//    constructing an XMPP URI or IRI [RFC5122].  In particular, such a
	//    character MUST be stripped before any other canonicalization steps
	//    are taken.

	domainpart = strings.TrimSuffix(domainpart, ".")

	return
}

func stringify(j JID) string {
	s := j.Domainpart()
	if lp := j.Localpart(); lp != "" {
		s = lp + "@" + s
	}
	if rp := j.Resourcepart(); rp != "" {
		s = s + "/" + rp
	}
	return s
}

func checkIP6String(domainpart string) error {
	// If the domainpart is a valid IPv6 address (with brackets), short circuit.
	if l := len(domainpart); l > 2 && strings.HasPrefix(domainpart, "[") &&
		strings.HasSuffix(domainpart, "]") {
		if ip := net.ParseIP(domainpart[1 : l-1]); ip == nil || ip.To4() != nil {
			return errors.New("Domainpart is not a valid IPv6 address")
		}
	}
	return nil
}

func commonChecks(localpart, domainpart, resourcepart string) error {
	l := len(localpart)
	if l > 1023 {
		return errors.New("The localpart must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.3.1 provides a small table of characters which are still not
	// allowed in localpart's even though the IdentifierClass base class and the
	// UsernameCaseMapped profile don't forbid them; disallow them here.
	if strings.ContainsAny(localpart, "\"&'/:<>@") {
		return errors.New("Localpart contains forbidden characters")
	}

	l = len(resourcepart)
	if l > 1023 {
		return errors.New("The resourcepart must be smaller than 1024 bytes")
	}

	l = len(domainpart)
	if l < 1 || l > 1023 {
		return errors.New("The domainpart must be between 1 and 1023 bytes")
	}

	if err := checkIP6String(domainpart); err != nil {
		return err
	}

	return nil
}
