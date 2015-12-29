// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
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

// UnsafeFromParts constructs a new UnsafeJID from the given localpart,
// domainpart, and resourcepart. The only required part is the domainpart
// ('example.net' and 'hostname' are valid Jids).
func UnsafeFromParts(localpart, domainpart, resourcepart string) (*UnsafeJID, error) {

	if err := commonChecks(localpart, domainpart, resourcepart); err != nil {
		return nil, err
	}

	return &UnsafeJID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *UnsafeJID) Bare() JID {
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
	return stringify(j)
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
