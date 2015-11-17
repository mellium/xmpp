// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

// Jid is an immutable representation of an XMPP address comprising a
// localpart, domainpart, and resourcepart.
type Jid struct {
	localpart    string
	domainpart   string
	resourcepart string
	validated    bool
}

func partsFromString(s string) (string, string, string, error) {
	var localpart, domainpart, resourcepart string

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
			return "", "", "", errors.New("The resourcepart must be larger than 0 bytes")
		}
	} else {
		resourcepart = ""
	}

	norp := strings.TrimSuffix(parts[0], "/")

	//    2.  Remove any portion from the beginning of the string to the first
	//        '@' character (if there is an '@' character present).

	nolp := strings.SplitAfterN(norp, "@", 2)

	if nolp[0] == "@" {
		return "", "", "", errors.New("The localpart must be larger than 0 bytes")
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

	return localpart, domainpart, resourcepart, nil
}

// FromString constructs a new Jid object from the given string representation.
// The string may be any valid bare or full JID including domain names, IP
// literals, or hosts.
func FromString(s string) (*Jid, error) {
	localpart, domainpart, resourcepart, err := partsFromString(s)
	if err != nil {
		return nil, err
	}
	return FromParts(localpart, domainpart, resourcepart)
}

// FromStringUnsafe constructs a Jid without performing any verification on the
// input string. This is unsafe and should only be used for trusted, internal
// data (eg. a Jid from the database that has already been validated). External
// data (user input, a JID sent over the wire via an XMPP connection, etc.)
// should use the FromString method instead.
func FromStringUnsafe(s string) (*Jid, error) {
	localpart, domainpart, resourcepart, err := partsFromString(s)
	if err != nil {
		return nil, err
	}
	return FromPartsUnsafe(localpart, domainpart, resourcepart), nil
}

// FromJid creates a new Jid object from an existing one. If the Jid has
// already been validated, FromJid(j) is identical to j.Copy(). Otherwise it
// validates the Jid and returns a new copy that has been validated and
// normalized. This means that Jid's created with FromJid will not necessarily
// be octet-for-octet identical with the old Jid and j.Equals(jid.FromJid(j))
// may return false.
func FromJid(j *Jid) (*Jid, error) {
	if j.validated {
		return j.Copy(), nil
	}

	return FromParts(j.localpart, j.domainpart, j.resourcepart)
}

// FromParts constructs a new Jid object from the given localpart, domainpart,
// and resourcepart. The only required part is the domainpart ('example.net'
// and 'hostname' are valid Jids).
func FromParts(localpart, domainpart, resourcepart string) (*Jid, error) {

	// Ensure that parts are valid UTF-8 (and short circuit the rest of the
	// process if they're not)
	if !utf8.ValidString(localpart) {
		return nil, errors.New("Localpart contains invalid UTF-8")
	}
	if !utf8.ValidString(resourcepart) {
		return nil, errors.New("Resourcepart contains invalid UTF-8")
	}

	// RFC 7622 §3.2.1:
	//
	//    An entity that prepares a string for inclusion in an XMPP domainpart
	//    slot MUST ensure that the string consists only of Unicode code points
	//    that are allowed in NR-LDH labels or U-labels as defined in
	//    [RFC5890].  This implies that the string MUST NOT include A-labels as
	//    defined in [RFC5890]; each A-label MUST be converted to a U-label
	//    during preparation of a string for inclusion in a domainpart slot.

	domainpart, err := idna.ToUnicode(domainpart)
	if err != nil {
		return nil, err
	}

	if !utf8.ValidString(domainpart) {
		return nil, errors.New("Domainpart contains invalid UTF-8")
	}

	// RFC 7622 §3.3:
	//
	//    The localpart of a JID is an instance of the UsernameCaseMapped
	//    profile of the PRECIS IdentifierClass, which is specified in
	//    [RFC7613].  The rules and considerations provided in that
	//    specification MUST be applied to XMPP localparts.
	//
	// RFC 7613 §3.2.1
	//
	//    An entity that prepares a string according to this profile MUST first
	//    map fullwidth and halfwidth characters to their decomposition
	//    mappings (see Unicode Standard Annex #11 [UAX11]).

	eastAsianNFKD := func(r rune) rune {
		if kind := width.LookupRune(r).Kind(); kind == width.EastAsianFullwidth ||
			kind == width.EastAsianHalfwidth {
			return []rune(norm.NFKD.String(string(r)))[0]
		}
		return r
	}
	localpart = strings.Map(eastAsianNFKD, localpart)

	// TODO:
	//
	//    After applying this width-mapping rule, the entity then MUST ensure
	//    that the string consists only of Unicode code points that conform to
	//    the PRECIS IdentifierClass defined in Section 4.2 of [RFC7564].

	// RFC 7613 §3.2.2
	//
	//    1.  Width-Mapping Rule: Applied as part of preparation (see above).
	//
	//    2.  Additional Mapping Rule: There is no additional mapping rule.
	//
	//    3.  Case-Mapping Rule: Uppercase and titlecase characters MUST be
	//        mapped to their lowercase equivalents, preferably using Unicode
	//        Default Case Folding as defined in the Unicode Standard [Unicode]
	//        (at the time of this writing, the algorithm is specified in
	//        Chapter 3 of [Unicode7.0], but the chapter number might change in
	//        a future version of the Unicode Standard); see further discussion
	//        in Section 3.4.

	lowercaser := cases.Lower(language.Und)
	localpart = lowercaser.String(localpart)

	//    4.  Normalization Rule: Unicode Normalization Form C (NFC) MUST be
	//        applied to all characters.

	localpart = norm.NFC.String(localpart)

	// TODO:
	//
	//    5.  Directionality Rule: Applications MUST apply the "Bidi Rule"
	//        defined in [RFC5893] to strings that contain right-to-left
	//        characters (i.e., each of the six conditions of the Bidi Rule
	//        must be satisfied).

	l := len(localpart)
	if l > 1023 {
		return nil, errors.New("The localpart must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.3.1 provides a small table of characters which are still not
	// allowed in localpart's even though the IdentifierClass base class and the
	// UsernameCaseMapped profile don't forbid them; remove them here.
	// TODO: Add XMPP-0106 support?
	if strings.ContainsAny(localpart, "\"&'/:<>@") {
		return nil, errors.New("Localpart contains forbidden characters")
	}

	// RFC 7622 §3.4:
	//
	//    The resourcepart of a JID is an instance of the OpaqueString profile
	//    of the PRECIS FreeformClass, which is specified in [RFC7613].  The
	//    rules and considerations provided in that specification MUST be
	//    applied to XMPP resourceparts.
	//
	// RFC 7613 §4.2.1.  Preparation
	//
	//    An entity that prepares a string according to this profile MUST
	//    ensure that the string consists only of Unicode code points that
	//    conform to the FreeformClass base string class defined in [RFC7564].
	//    In addition, the entity MUST encode the string as UTF-8 [RFC3629].

	// [TODO]

	// RFC 7613 §4.2.2.  Enforcement
	//
	//    An entity that performs enforcement according to this profile MUST
	//    prepare a string as described in Section 4.2.1 and MUST also apply
	//    the rules specified below for the OpaqueString profile (these rules
	//    MUST be applied in the order shown):
	//
	//    1.  Width-Mapping Rule: Fullwidth and halfwidth characters MUST NOT
	//        be mapped to their decomposition mappings (see Unicode Standard
	//        Annex #11 [UAX11]).
	//
	//    2.  Additional Mapping Rule: Any instances of non-ASCII space MUST be
	//        mapped to ASCII space (U+0020); a non-ASCII space is any Unicode
	//        code point having a Unicode general category of "Zs" (with the
	//        exception of U+0020).

	resourcepart = strings.Map(func(r rune) rune {
		if unicode.In(r, unicode.Zs) {
			return '\u0020'
		}
		return r
	}, resourcepart)

	//    3.  Case-Mapping Rule: Uppercase and titlecase characters MUST NOT be
	//        mapped to their lowercase equivalents.
	//
	//    4.  Normalization Rule: Unicode Normalization Form C (NFC) MUST be
	//        applied to all characters.

	resourcepart = norm.NFC.String(resourcepart)

	l = len(resourcepart)
	if l > 1023 {
		return nil, errors.New("The resourcepart must be smaller than 1024 bytes")
	}

	// RFC 7622 §3.2.2:
	//
	//    An entity that performs enforcement in XMPP domainpart slots MUST
	//    prepare a string as described in Section 3.2.1 and MUST also apply
	//    the normalization, case-mapping, and width-mapping rules defined in
	//    [RFC5892].
	//
	// I'm assuming that the reference to RFC 5892 is wrong, and that it meant
	// RFC 5895.
	//
	// RFC 5895 §2. The General Procedure:
	//
	//    1.  Uppercase characters are mapped to their lowercase equivalents by
	//        using the algorithm for mapping case in Unicode characters.  This
	//        step was chosen because the output will behave more like ASCII
	//        host names behave.

	domainpart = lowercaser.String(domainpart)

	//    2.  Fullwidth and halfwidth characters (those defined with
	//        Decomposition Types <wide> and <narrow>) are mapped to their
	//        decomposition mappings as shown in the Unicode character
	//        database.  This step was chosen because many input mechanisms,
	//        particularly in Asia, do not allow you to easily enter characters
	//        in the form used by IDNA2008.  Even if they do allow the correct
	//        character form, the user might not know which form they are
	//        entering.

	domainpart = strings.Map(eastAsianNFKD, domainpart)

	//    3.  All characters are mapped using Unicode Normalization Form C
	//        (NFC).  This step was chosen because it maps combinations of
	//        combining characters into canonical composed form.  As with the
	//        fullwidth/halfwidth mapping, users are not generally aware of the
	//        particular form of characters that they are entering, and
	//        IDNA2008 requires that only the canonical composed forms from NFC
	//        be used.

	domainpart = norm.NFC.String(domainpart)

	//    4.  [IDNA2008protocol] is specified such that the protocol acts on
	//        the individual labels of the domain name.  If an implementation
	//        of this mapping is also performing the step of separation of the
	//        parts of a domain name into labels by using the FULL STOP
	//        character (U+002E), the IDEOGRAPHIC FULL STOP character (U+3002)
	//        can be mapped to the FULL STOP before label separation occurs.
	//        There are other characters that are used as "full stops" that one
	//        could consider mapping as label separators, but their use as such
	//        has not been investigated thoroughly.  This step was chosen
	//        because some input mechanisms do not allow the user to easily
	//        enter proper label separators.  Only the IDEOGRAPHIC FULL STOP
	//        character (U+3002) is added in this mapping because the authors
	//        have not fully investigated the applicability of other characters
	//        and the environments where they should and should not be
	//        considered domain name label separators.
	//
	// We'll go ahead and do this for comparison purposes:

	domainpart = strings.Replace(domainpart, "\u3002", ".", -1)

	l = len(domainpart)
	if l < 1 || l > 1023 {
		return nil, errors.New("The domainpart must be between 1 and 1023 bytes")
	}

	return &Jid{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
		validated:    true,
	}, nil
}

// FromPartsUnsafe constructs a Jid from the given localpart, domainpart, and
// resourcepart without performing any verification on the
// input string.
func FromPartsUnsafe(localpart, domainpart, resourcepart string) *Jid {
	return &Jid{
		localpart:    localpart,
		domainpart:   domainpart,
		resourcepart: resourcepart,
		validated:    false,
	}
}

// Bare returns a copy of the Jid without a resourcepart. This is sometimes
// called a "bare" JID.
func (j *Jid) Bare() *Jid {
	return &Jid{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: "",
		validated:    j.validated,
	}
}

// IsBare is true if the JID is a bare JID (it has no resourcepart).
func (j *Jid) IsBare() bool {
	return j.resourcepart == ""
}

// Localpart gets the localpart of a JID (eg "username").
func (j *Jid) Localpart() string {
	return j.localpart
}

// Domainpart gets the domainpart of a JID (eg. "example.net").
func (j *Jid) Domainpart() string {
	return j.domainpart
}

// Resourcepart gets the resourcepart of a JID (eg. "mobile").
func (j *Jid) Resourcepart() string {
	return j.resourcepart
}

// Checks if the given Jid object has been validated and normalized.
func (j *Jid) Validated() bool {
	return j.validated
}

// Makes an identical (octet-for-octet) copy of the given Jid.
// j.Equals(j.Copy()) will always return true.
func (j *Jid) Copy() *Jid {
	return &Jid{
		localpart:    j.localpart,
		domainpart:   j.domainpart,
		resourcepart: j.resourcepart,
		validated:    j.validated,
	}
}

func (j *Jid) Equals(j2 *Jid) bool {
	return j.localpart == j2.localpart &&
		j.domainpart == j2.domainpart && j.resourcepart == j2.resourcepart
}

// String converts a `Jid` object to its string representation.
func (j *Jid) String() string {
	s := j.domainpart
	if j.localpart != "" {
		s = j.localpart + "@" + s
	}
	if j.resourcepart != "" {
		s = s + "/" + j.resourcepart
	}
	return s
}

// MarshalXMLAttr marshals the JID as an XML attribute for use with the
// encoding/xml package.
func (j *Jid) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: j.String()}, nil
}
