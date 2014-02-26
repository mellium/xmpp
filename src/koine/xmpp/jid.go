// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// TODO:
//  - Make domainpart the only required part of a JID
//  - Validate that domainpart is a valid IP-literal, IPv4address, or ifqdn
//  - If using an IPv6 address, surround with brackets
//  - Verify that domains are > 0 bytes and < 1023 bytes in length

// Package jid implements XMPP addresses (JIDs) as described in RFC 6122. The
// syntax for a JID is defined as follows using the Augmented Backus-Naur Form:
//
//      jid           = [ localpart "@" ] domainpart [ "/" resourcepart ]
//      localpart     = 1*(nodepoint)
//                      ;
//                      ; a "nodepoint" is a UTF-8 encoded Unicode code
//                      ; point that satisfies the Nodeprep profile of
//                      ; stringprep
//                      ;
//      domainpart    = IP-literal / IPv4address / ifqdn
//                      ;
//                      ; the "IPv4address" and "IP-literal" rules are
//                      ; defined in RFC 3986, and the first-match-wins
//                      ; (a.k.a. "greedy") algorithm described in RFC
//                      ; 3986 applies to the matching process
//                      ;
//                      ; note well that reuse of the IP-literal rule
//                      ; from RFC 3986 implies that IPv6 addresses are
//                      ; enclosed in square brackets (i.e., beginning
//                      ; with '[' and ending with ']'), which was not
//                      ; the case in RFC 3920
//                      ;
//      ifqdn         = 1*(namepoint)
//                      ;
//                      ; a "namepoint" is a UTF-8 encoded Unicode
//                      ; code point that satisfies the Nameprep
//                      ; profile of stringprep
//                      ;
//      resourcepart  = 1*(resourcepoint)
//                      ;
//                      ; a "resourcepoint" is a UTF-8 encoded Unicode
//                      ; code point that satisfies the Resourceprep
//                      ; profile of stringprep
//                      ;
package jid

import (
	"code.google.com/p/go.text/unicode/norm"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Define some reusable error messages.
const (
	ERROR_INVALID_STRING = "String is not valid UTF-8"
	ERROR_INVALID_JID    = "String is not a valid JID"
)

// The Unicode form that everything should be normalized with.
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
func (address jid) LocalPart() string {
	return address.localpart
}

// Get the domainpart of a JID
func (address jid) DomainPart() string {
	return address.domainpart
}

// Get the resourcepart of a JID
func (address jid) ResourcePart() string {
	return address.resourcepart
}

// Set the localpart of a JID and verify that it is a valid/normalized UTF-8
// string.
func (address jid) SetLocalPart(localpart string) error {
	if utf8.ValidString(localpart) {
		address.localpart = NF.String(localpart)
		return nil
	}
	return errors.New(ERROR_INVALID_STRING)
}

// Set the domainpart of a JID and verify that it is a valid/normalized  UTF-8
// string.
func (address jid) SetDomainPart(domainpart string) error {
	if utf8.ValidString(domainpart) {
		address.domainpart = NF.String(domainpart)
		return nil
	}
	return errors.New(ERROR_INVALID_STRING)
}

// Set the resourcepart of a JID and verify that it is a valid/normalized UTF-8
// string.
func (address jid) SetResourcePart(resourcepart string) error {
	if utf8.ValidString(resourcepart) {
		address.resourcepart = NF.String(resourcepart)
		return nil
	}
	return errors.New(ERROR_INVALID_STRING)
}

// Return the full JID as a string
func (address jid) String() string {
	return address.LocalPart() + "@" + address.DomainPart() + "/" + address.ResourcePart()
}

// Set the JIDs properties from a string.
const JIDMatch = "[^@/]+@[^@/]+/[^@/]+"

func (address jid) FromString(s string) error {
	// Make sure the string is valid UTF-8
	if !utf8.ValidString(s) {
		return errors.New(ERROR_INVALID_STRING)
	}
	// Normalize the UTF-8 sequence and make sure it is a valid JID
	normalized := strings.TrimSpace(NF.String(s))
	switch matched, err := regexp.MatchString(JIDMatch, normalized); {
	case err != nil:
		return err
	case !matched:
		return errors.New(ERROR_INVALID_JID)
	}
	// Set the various parts of the JID
	atLoc := strings.IndexRune(normalized, '@')
	slashLoc := strings.IndexRune(normalized, '/')
	address.SetLocalPart(normalized[0:atLoc])
	address.SetDomainPart(normalized[atLoc+1 : slashLoc])
	address.SetResourcePart(normalized[slashLoc+1:])
	return nil
}
