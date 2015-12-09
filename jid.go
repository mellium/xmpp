// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"strings"
)

// JID defines methods that are common to all XMPP address (historically,
// "Jabber ID") implementations.
type JID interface {
	Localpart() string
	Domainpart() string
	Resourcepart() string

	String() string
	Equal(other JID) bool

	xml.MarshalerAttr
	xml.UnmarshalerAttr
}

// SplitString splits out the localpart, domainpart, and resourcepart from a
// string representation of a JID. The parts are not guaranteed to be valid, and
// SplitString only performs basic length validation on the individual parts.
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
