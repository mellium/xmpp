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

// EnforcedJID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart, that has undergone the PRECIS preparation and
// enforcement steps. This is normally what servers will want to use for all
// internal JID's.
type EnforcedJID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// EnforcedFromString constructs a new EnforcedJID from the given string
// representation.
func EnforcedFromString(s string) (*EnforcedJID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return EnforcedFromParts(localpart, domainpart, resourcepart)
}

// EnforcedFromParts constructs a new EnforcedJID from the given localpart,
// domainpart, and resourcepart.
func EnforcedFromParts(localpart, domainpart, resourcepart string) (*EnforcedJID, error) {
	// Ensure that parts are valid UTF-8 (and short circuit the rest of the
	// process if they're not). We'll check the domainpart after performing
	// the IDNA ToUnicode operation.
	if !utf8.ValidString(localpart) || !utf8.ValidString(resourcepart) {
		return nil, errors.New("JID contains invalid UTF-8")
	}

	domainpart, err := idna.ToUnicode(domainpart)
	if err != nil {
		return nil, err
	}

	if !utf8.ValidString(domainpart) {
		return nil, errors.New("Domainpart contains invalid UTF-8")
	}

	// TODO: Do enforcement properly

	if err := commonChecks(localpart, domainpart, resourcepart); err != nil {
		return nil, err
	}

	return &EnforcedJID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *EnforcedJID) Bare() *EnforcedJID {
	return &EnforcedJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *EnforcedJID) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *EnforcedJID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *EnforcedJID) Resourcepart() string {
	return j.resourcepart
}

// Makes a copy of the given Jid. j.Equals(j.Copy()) will always return true.
func (j *EnforcedJID) Copy() *EnforcedJID {
	return &EnforcedJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
	}
}

// String converts an EnforcedJID to its string representation.
func (j *EnforcedJID) String() string {
	return stringify(j)
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *EnforcedJID) Equal(j2 JID) bool {
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the JID as
// an XML attribute.
func (j *EnforcedJID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid JID (or returns an error).
func (j *EnforcedJID) UnmarshalXMLAttr(attr xml.Attr) error {
	jid, err := EnforcedFromString(attr.Value)
	j.localpart = jid.Localpart()
	j.domainpart = jid.Domainpart()
	j.resourcepart = jid.Resourcepart()
	return err
}
