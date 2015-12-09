// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// PreparedJID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart, that has undergone the PRECIS preparation step.
// This is normally what clients will want to use for all internal JID's.
type PreparedJID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// PreparedFromString constructs a new PreparedJID from the given string
// representation.
func PreparedFromString(s string) (*PreparedJID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return PreparedFromParts(localpart, domainpart, resourcepart)
}

// PreparedFromParts constructs a new PreparedJID from the given localpart,
// domainpart, and resourcepart.
func PreparedFromParts(localpart, domainpart, resourcepart string) (*PreparedJID, error) {
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

	// TODO: Do preparation properly

	if err := commonChecks(localpart, domainpart, resourcepart); err != nil {
		return nil, err
	}

	return &PreparedJID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *PreparedJID) Bare() *PreparedJID {
	return &PreparedJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *PreparedJID) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *PreparedJID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *PreparedJID) Resourcepart() string {
	return j.resourcepart
}

// Makes a copy of the given Jid. j.Equals(j.Copy()) will always return true.
func (j *PreparedJID) Copy() *PreparedJID {
	return &PreparedJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
	}
}

// String converts an PreparedJID to its string representation.
func (j *PreparedJID) String() string {
	return stringify(j)
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *PreparedJID) Equal(j2 JID) bool {
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the JID as
// an XML attribute.
func (j *PreparedJID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid JID (or returns an error).
func (j *PreparedJID) UnmarshalXMLAttr(attr xml.Attr) error {
	jid, err := PreparedFromString(attr.Value)
	j.localpart = jid.Localpart()
	j.domainpart = jid.Domainpart()
	j.resourcepart = jid.Resourcepart()
	return err
}
