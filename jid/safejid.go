// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/precis"
)

// SafeJID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart, that is safe to display in a user interface,
// send over the wire, or compare with another SafeJID. All parts of a safe JID
// are guaranteed to be valid UTF-8 and will be represented in their canonical
// form which gives comparison the greatest chance of succeeding.
type SafeJID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// SafeFromString constructs a new SafeJID from the given string
// representation.
func SafeFromString(s string) (*SafeJID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return SafeFromParts(localpart, domainpart, resourcepart)
}

// SafeFromParts constructs a new SafeJID from the given localpart,
// domainpart, and resourcepart.
func SafeFromParts(localpart, domainpart, resourcepart string) (*SafeJID, error) {
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

	var err error
	domainpart, err = idna.ToUnicode(domainpart)
	if err != nil {
		return nil, err
	}

	if !utf8.ValidString(domainpart) {
		return nil, errors.New("Domainpart contains invalid UTF-8")
	}

	// RFC 7622 §3.2.2.  Enforcement
	//
	//   An entity that performs enforcement in XMPP domainpart slots MUST
	//   prepare a string as described in Section 3.2.1 and MUST also apply
	//   the normalization, case-mapping, and width-mapping rules defined in
	//   [RFC5892].
	//
	// TODO: I have no idea what this is talking about… what rules? RFC 5892 is a
	//       bunch of property lists. Maybe it meant RFC 5895?

	localpart, err = precis.UsernameCaseMapped.String(localpart)
	if err != nil {
		return nil, err
	}

	resourcepart, err = precis.OpaqueString.String(resourcepart)
	if err != nil {
		return nil, err
	}

	if err := commonChecks(localpart, domainpart, resourcepart); err != nil {
		return nil, err
	}

	return &SafeJID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *SafeJID) Bare() JID {
	return &SafeJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *SafeJID) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *SafeJID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *SafeJID) Resourcepart() string {
	return j.resourcepart
}

// Makes a copy of the given Jid. j.Equal(j.Copy()) will always return true.
func (j *SafeJID) Copy() *SafeJID {
	return &SafeJID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
	}
}

// String converts an SafeJID to its string representation.
func (j *SafeJID) String() string {
	return stringify(j)
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *SafeJID) Equal(j2 JID) bool {
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the JID as
// an XML attribute.
func (j *SafeJID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid JID (or returns an error).
func (j *SafeJID) UnmarshalXMLAttr(attr xml.Attr) error {
	jid, err := SafeFromString(attr.Value)
	j.localpart = jid.Localpart()
	j.domainpart = jid.Domainpart()
	j.resourcepart = jid.Resourcepart()
	return err
}
