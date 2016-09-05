// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/secure/precis"
)

const escape = ` "&'/:<>@\`

func shouldEscape(c byte) bool {
	return c == ' ' || c == '"' || c == '&' || c == '\'' || c == '/' || c == ':' || c == '<' || c == '>' || c == '@' || c == '\\'
}

// I just wrote these all out because it's a lot faster and not likely to
// change; is it really worth the confusing logic though?
func shouldUnescape(s string) bool {
	return (s[0] == '2' && (s[1] == '0' || s[1] == '2' || s[1] == '6' || s[1] == '7' || s[1] == 'f' || s[1] == 'F')) || (s[0] == '3' && (s[1] == 'a' || s[1] == 'A' || s[1] == 'c' || s[1] == 'C' || s[1] == 'e' || s[1] == 'E')) || (s[0] == '4' && s[1] == '0') || (s[0] == '5' && (s[1] == 'c' || s[1] == 'C'))
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// BUG(ssw): Unescape does not fail on invalid escape codes.

// Unescape returns an unescaped version of the specified localpart using the
// escaping mechanism defined in XEP-0106: JID Escaping. It only unescapes
// sequences documented in XEP-0106 and does not guarantee that the resulting
// localpart is well formed.
func Unescape(s string) string {
	n := 0
	for i := 0; i < len(s); i++ {
		if len(s) < i+3 {
			break
		}
		if s[i] == '\\' && shouldUnescape(s[i+1:i+3]) {
			n++
			i += 2
		}
	}

	if n == 0 {
		return s
	}

	t := make([]byte, len(s)-2*n)
	j := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && len(s) > i+2 && shouldUnescape(s[i+1:i+3]) {
			t[j] = unhex(s[i+1])<<4 | unhex(s[i+2])
			i += 2
		} else {
			t[j] = s[i]
		}
		j++
	}
	return string(t)
}

// Escape returns an escaped version of the specified localpart using the
// escaping mechanism defined in XEP-0106: JID Escaping. It is not applied
// by any of the JID methods, and must be applied manually before constructing a
// JID.
func Escape(s string) string {
	count := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			count++
		}
	}

	if count == 0 {
		return s
	}

	t := make([]byte, len(s)+2*count)
	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case shouldEscape(c):
			t[j] = '\\'
			t[j+1] = "0123456789abcdef"[c>>4]
			t[j+2] = "0123456789abcdef"[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

// JID represents an XMPP address (Jabber ID) comprising a localpart,
// domainpart, and resourcepart. All parts of a JID are guaranteed to be valid
// UTF-8 and will be represented in their canonical form which gives comparison
// the greatest chance of succeeding.
type JID struct {
	localpart    string
	domainpart   string
	resourcepart string
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
	if j == nil {
		return j
	}

	return &JID{
		localpart:    "",
		domainpart:   j.domainpart,
		resourcepart: "",
	}
}

// Localpart gets the localpart of a JID (eg "username").
func (j *JID) Localpart() string {
	if j == nil {
		return ""
	}
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *JID) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "someclient-abc123").
func (j *JID) Resourcepart() string {
	if j == nil {
		return ""
	}
	return j.resourcepart
}

// Copy makes a copy of the given Jid. j.Equal(j.Copy()) will always return
// true.
func (j *JID) Copy() *JID {
	if j == nil {
		return j
	}

	return &JID{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
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
func (j *JID) Equal(j2 *JID) bool {
	if j == nil || j2 == nil {
		return j == j2
	}
	return j.Localpart() == j2.Localpart() &&
		j.Domainpart() == j2.Domainpart() && j.Resourcepart() == j2.Resourcepart()
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
		j.localpart = j2.localpart
		j.domainpart = j2.domainpart
		j.resourcepart = j2.resourcepart
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
	if strings.ContainsAny(localpart, `"&'/:<>@`) {
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
