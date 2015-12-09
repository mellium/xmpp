// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"net"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// UnsafeJID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart, that is not Unicode safe.
type UnsafeJID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// UnsafeFromString constructs a new UnsafeJID from the given string
// representation. The string may be any valid bare or full JID including
// raw domain names, IP literals, or hosts.
func UnsafeFromString(s string) (*UnsafeJID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return UnsafeFromParts(localpart, domainpart, resourcepart)
}

// UnsafeFromParts constructs a new UnsafeJID object from the given localpart,
// domainpart, and resourcepart. The only required part is the domainpart
// ('example.net' and 'hostname' are valid Jids). JID's which include IDNA
// A-labels in the domainpart will be converted to their Unicode representation.
func UnsafeFromParts(localpart, domainpart, resourcepart string) (*UnsafeJID, error) {

	// Ensure that parts are valid UTF-8 (and short circuit the rest of the
	// process if they're not). We'll check the domainpart after performing
	// the IDNA ToUnicode operation.
	if !utf8.ValidString(localpart) || !utf8.ValidString(resourcepart) {
		return nil, errors.New("JID contains invalid UTF-8")
	}

	// RFC 7622 §3.2.1.  Preparation
	//
	//    An entity that prepares a string for inclusion in an XMPP domainpart
	//    slot MUST ensure that the string consists only of Unicode code points
	//    that are allowed in NR-LDH labels or U-labels as defined in
	//    [RFC5890].  This implies that the string MUST NOT include A-labels as
	//    defined in [RFC5890]; each A-label MUST be converted to a U-label
	//    during preparation of a string for inclusion in a domainpart slot.
	//
	// While we're not doing preparation yet, we're also going to store all JIDs
	// as Unicode strings, so let's go ahead and do this (even for Unsafe JID's).

	domainpart, err := idna.ToUnicode(domainpart)
	if err != nil {
		return nil, err
	}

	if !utf8.ValidString(domainpart) {
		return nil, errors.New("Domainpart contains invalid UTF-8")
	}

	l := len(localpart)
	if l > 1023 {
		return nil, errors.New("The localpart must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.3.1 provides a small table of characters which are still not
	// allowed in localpart's even though the IdentifierClass base class and the
	// UsernameCaseMapped profile don't forbid them; remove them here.
	if strings.ContainsAny(localpart, "\"&'/:<>@") {
		return nil, errors.New("Localpart contains forbidden characters")
	}

	l = len(resourcepart)
	if l > 1023 {
		return nil, errors.New("The resourcepart must be smaller than 1024 bytes")
	}

	l = len(domainpart)
	if l < 1 || l > 1023 {
		return nil, errors.New("The domainpart must be between 1 and 1023 bytes")
	}

	// If the domainpart is a valid IPv6 address (with brackets), short circuit.
	if l := len(domainpart); l > 2 && strings.HasPrefix(domainpart, "[") &&
		strings.HasSuffix(domainpart, "]") {
		if ip := net.ParseIP(domainpart[1 : l-1]); ip != nil && ip.To4() == nil {
			return &UnsafeJID{
				localpart:    localpart,
				domainpart:   domainpart,
				resourcepart: resourcepart,
			}, nil
		} else {
			// If the domainpart has brackets, but is not an IPv6 address, error.
			return nil, errors.New("Domainpart is not a valid IPv6 address")
		}
	}

	// If the domainpart is a valid IPv4 address, short circuit.
	if ip := net.ParseIP(domainpart); ip != nil && ip.To4() != nil {
		return &UnsafeJID{
			localpart:    localpart,
			domainpart:   domainpart,
			resourcepart: resourcepart,
		}, nil
	}

	return &UnsafeJID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *UnsafeJID) Bare() *UnsafeJID {
	return &UnsafeJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *UnsafeJID) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *UnsafeJID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *UnsafeJID) Resourcepart() string {
	return j.resourcepart
}

// Makes a copy of the given Jid. j.Equals(j.Copy()) will always return true.
func (j *UnsafeJID) Copy() *UnsafeJID {
	return &UnsafeJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
	}
}

// String converts an UnsafeJID to its string representation.
func (j *UnsafeJID) String() string {
	s := j.Domainpart()
	if lp := j.Localpart(); lp != "" {
		s = lp + "@" + s
	}
	if rp := j.Resourcepart(); rp != "" {
		s = s + "/" + rp
	}
	return s
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *UnsafeJID) Equal(j2 JID) bool {
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the JID as
// an XML attribute.
func (j *UnsafeJID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid JID (or returns an error).
func (j *UnsafeJID) UnmarshalXMLAttr(attr xml.Attr) error {
	jid, err := UnsafeFromString(attr.Value)
	j.localpart = jid.Localpart()
	j.domainpart = jid.Domainpart()
	j.resourcepart = jid.Resourcepart()
	return err
}
