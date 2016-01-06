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
	"golang.org/x/text/unicode/precis"
)

// JID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart. All parts of a JID are guaranteed to be valid
// UTF-8 and will be represented in their canonical form which gives comparison
// the greatest chance of succeeding unless any of the "unsafe" methods are used
// to create the JID.
type JID struct {
	localpart    string
	domainpart   string
	resourcepart string
}

// ParseString constructs a new JID from the given string
// representation.
func ParseString(s string) (*JID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return New(localpart, domainpart, resourcepart)
}

// New constructs a new JID from the given localpart, domainpart, and
// resourcepart.
func New(localpart, domainpart, resourcepart string) (*JID, error) {
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

	if localpart != "" {
		localpart, err = precis.UsernameCaseMapped.String(localpart)
		if err != nil {
			return nil, err
		}
	}

	if resourcepart != "" {
		resourcepart, err = precis.OpaqueString.String(resourcepart)
		if err != nil {
			return nil, err
		}
	}

	if err := commonChecks(localpart, domainpart, resourcepart); err != nil {
		return nil, err
	}

	return &JID{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
	}, nil
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *JID) Bare() *JID {
	return &JID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Domain returns a copy of the Jid without a resourcepart or localpart.
func (j *JID) Domain() *JID {
	return &JID{
		localpart:    "",
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *JID) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *JID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *JID) Resourcepart() string {
	return j.resourcepart
}

// Makes a copy of the given Jid. j.Equal(j.Copy()) will always return true.
func (j *JID) Copy() *JID {
	return &JID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
	}
}

// String converts an JID to its string representation.
func (j JID) String() string {
	s := j.domainpart
	if j.localpart != "" {
		s = j.localpart + "@" + s
	}
	if j.resourcepart != "" {
		s = s + "/" + j.resourcepart
	}
	return s
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *JID) Equal(j2 JID) bool {
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the JID as
// an XML attribute.
func (j JID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid JID (or returns an error).
func (j *JID) UnmarshalXMLAttr(attr xml.Attr) error {
	jid, err := ParseString(attr.Value)
	j.localpart = jid.localpart
	j.domainpart = jid.domainpart
	j.resourcepart = jid.resourcepart
	return err
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
