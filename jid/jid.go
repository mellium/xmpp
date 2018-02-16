// Copyright 2014 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid

import (
	"bytes"
	"encoding/xml"
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/secure/precis"
)

// JID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart. All parts of a JID are guaranteed to be valid
// UTF-8 and will be represented in their canonical form which gives comparison
// the greatest chance of succeeding.
type JID struct {
	locallen  int
	domainlen int
	data      []byte
}

// Parse constructs a new JID from the given string representation.
func Parse(s string) (*JID, error) {
	localpart, domainpart, resourcepart, err := SplitString(s)
	if err != nil {
		return nil, err
	}
	return New(localpart, domainpart, resourcepart)
}

// MustParse is like Parse but panics if the JID cannot be parsed.
// It simplifies safe initialization of JIDs from known-good constant strings.
func MustParse(s string) *JID {
	j, err := Parse(s)
	if err != nil {
		if strconv.CanBackquote(s) {
			s = "`" + s + "`"
		} else {
			s = strconv.Quote(s)
		}
		panic(`jid: Parse(` + s + `): ` + err.Error())
	}
	return j
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

	var lenlocal int
	data := make([]byte, 0, len(localpart)+len(domainpart)+len(resourcepart))

	if localpart != "" {
		data, err = precis.UsernameCaseMapped.Append(data, []byte(localpart))
		if err != nil {
			return nil, err
		}
		lenlocal = len(data)
	}

	data = append(data, []byte(domainpart)...)

	if resourcepart != "" {
		data, err = precis.OpaqueString.Append(data, []byte(resourcepart))
		if err != nil {
			return nil, err
		}
	}

	if err := commonChecks(data[:lenlocal], domainpart, data[lenlocal+len(domainpart):]); err != nil {
		return nil, err
	}

	return &JID{
		locallen:  lenlocal,
		domainlen: len(domainpart),
		data:      data,
	}, nil
}

// WithResource returns a copy of the JID with a new resourcepart.
// This elides validation of the localpart and domainpart.
func (j *JID) WithResource(resourcepart string) (*JID, error) {
	var err error
	new := j.Bare()
	data := make([]byte, len(new.data), len(new.data)+len(resourcepart))
	copy(data, new.data)
	if resourcepart != "" {
		if !utf8.ValidString(resourcepart) {
			return nil, errors.New("JID contains invalid UTF-8")
		}
		data, err = precis.OpaqueString.Append(data, []byte(resourcepart))
		new.data = data
	}
	return new, err
}

// Bare returns a copy of the JID without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *JID) Bare() *JID {
	return &JID{
		locallen:  j.locallen,
		domainlen: j.domainlen,
		data:      j.data[:j.domainlen+j.locallen],
	}
}

// Domain returns a copy of the JID without a resourcepart or localpart.
func (j *JID) Domain() *JID {
	if j == nil {
		return j
	}

	return &JID{
		domainlen: j.domainlen,
		data:      j.data[j.locallen : j.domainlen+j.locallen],
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *JID) Localpart() string {
	if j == nil {
		return ""
	}
	return string(j.data[:j.locallen])
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *JID) Domainpart() string {
	return string(j.data[j.locallen : j.locallen+j.domainlen])
}

// Resourcepart gets the resourcepart of a JID.
func (j *JID) Resourcepart() string {
	if j == nil {
		return ""
	}
	return string(j.data[j.locallen+j.domainlen:])
}

// Copy makes a copy of the given JID. j.Equal(j.Copy()) will always return
// true.
func (j *JID) Copy() *JID {
	if j == nil {
		return j
	}

	return &JID{
		locallen:  j.locallen,
		domainlen: j.domainlen,
		data:      j.data,
	}
}

// Network satisfies the net.Addr interface by returning the name of the network
// ("xmpp").
func (*JID) Network() string {
	return "xmpp"
}

// String converts an JID to its string representation.
func (j *JID) String() string {
	if j == nil {
		return ""
	}
	s := string(j.data[j.locallen : j.locallen+j.domainlen])
	var addsep int
	if j.locallen > 0 {
		s = string(j.data[:j.locallen]) + "@" + s
		addsep = 1
	}
	if len(s) != len(j.data)+addsep {
		s = s + "/" + string(j.data[j.locallen+j.domainlen:])
	}
	return s
}

// Equal performs an octet-for-octet comparison with the given JID.
func (j *JID) Equal(j2 *JID) bool {
	if j == nil || j2 == nil {
		return j == j2
	}
	if len(j.data) != len(j2.data) {
		return false
	}
	for i := 0; i < len(j.data); i++ {
		if j.data[i] != j2.data[i] {
			return false
		}
	}
	return j.locallen == j2.locallen && j.domainlen == j2.domainlen
}

// MarshalXML satisfies the xml.Marshaler interface and marshals the JID as
// XML chardata.
func (j *JID) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	if err = e.EncodeToken(start); err != nil {
		return
	}
	if err = e.EncodeToken(xml.CharData(j.String())); err != nil {
		return
	}
	if err = e.EncodeToken(start.End()); err != nil {
		return
	}
	err = e.Flush()
	return
}

// UnmarshalXML satisfies the xml.Unmarshaler interface and unmarshals the JID
// from the elements chardata.
func (j *JID) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	data := struct {
		CharData string `xml:",chardata"`
	}{}
	if err = d.DecodeElement(&data, &start); err != nil {
		return
	}
	j2, err := Parse(data.CharData)

	if err == nil {
		j.locallen = j2.locallen
		j.domainlen = j2.domainlen
		j.data = j2.data
	}

	return
}

// MarshalXMLAttr satisfies the xml.MarshalerAttr interface and marshals the JID
// as an XML attribute.
func (j *JID) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if j == nil {
		return xml.Attr{}, nil
	}
	return xml.Attr{Name: name, Value: j.String()}, nil
}

// UnmarshalXMLAttr satisfies the xml.UnmarshalerAttr interface and unmarshals
// an XML attribute into a valid JID (or returns an error).
func (j *JID) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "" {
		return nil
	}
	jid, err := Parse(attr.Value)
	j.locallen = jid.locallen
	j.domainlen = jid.domainlen
	j.data = jid.data
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
	sep := strings.Index(s, "/")

	if sep == -1 {
		resourcepart = ""
	} else {
		// If the resource part exists, make sure it isn't empty.
		if sep == len(s)-1 {
			err = errors.New("The resourcepart must be larger than 0 bytes")
			return
		}
		resourcepart = s[sep+1:]
		s = s[:sep]
	}

	//    2.  Remove any portion from the beginning of the string to the first
	//        '@' character (if there is an '@' character present).

	sep = strings.Index(s, "@")

	switch sep {
	case -1:
		// There is no @ sign, and therefore no localpart.
		localpart = ""
		domainpart = s
	case 0:
		// The JID starts with an @ sign (invalid empty localpart)
		err = errors.New("The localpart must be larger than 0 bytes")
		return
	default:
		domainpart = s[sep+1:]
		localpart = s[:sep]
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

func commonChecks(localpart []byte, domainpart string, resourcepart []byte) error {
	l := len(localpart)
	if l > 1023 {
		return errors.New("The localpart must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.3.1 provides a small table of characters which are still not
	// allowed in localpart's even though the IdentifierClass base class and the
	// UsernameCaseMapped profile don't forbid them; disallow them here.
	if bytes.ContainsAny(localpart, `"&'/:<>@`) {
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
