// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version is a version of XMPP.
type Version struct {
	Major uint8
	Minor uint8
}

// ParseVersion parses a string of the form "Major.Minor" into a Version struct
// or returns an error.
func ParseVersion(s string) (Version, error) {
	v := Version{}

	versions := strings.Split(s, ".")
	if len(versions) != 2 {
		return v, errors.New("XMPP version must have a single separator.")
	}

	// Parse major version number
	major, err := strconv.ParseUint(versions[0], 10, 8)
	if err != nil {
		return v, err
	}
	v.Major = uint8(major)

	// Parse minor version number
	minor, err := strconv.ParseUint(versions[1], 10, 8)
	if err != nil {
		return v, err
	}
	v.Minor = uint8(minor)

	return v, nil
}

// Prints a string representation of the XMPP version in the form "Major.Minor".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// MarshalXMLAttr satisfies the MarshalerAttr interface and marshals the version
// as an XML attribute using its string representation.
func (v Version) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: v.String()}, nil
}

// UnmarshalXMLAttr satisfies the UnmarshalerAttr interface and unmarshals an
// XML attribute into a valid XMPP version (or returns an error).
func (v *Version) UnmarshalXMLAttr(attr xml.Attr) error {
	newVersion, err := ParseVersion(attr.Value)
	if err != nil {
		return err
	}

	v.Major = newVersion.Major
	v.Minor = newVersion.Minor
	return nil
}
