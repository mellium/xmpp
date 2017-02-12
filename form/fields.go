// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package form

import (
	"encoding/xml"
)

// A field represents a data field that may be added to a form.
type field struct {
	XMLName  xml.Name   `xml:"jabber:x:data field"`
	Typ      string     `xml:"type,attr"`
	Var      string     `xml:"var,attr,omitempty"`
	Label    string     `xml:"label,attr,omitempty"`
	Desc     string     `xml:"desc,omitempty"`
	Value    []string   `xml:"value,omitempty"`
	Required struct{}   `xml:"required,omitempty"`
	Option   []fieldopt `xml:"option,omitempty"`
}

type fieldopt struct {
	XMLName xml.Name `xml:"jabber:x:data option"`
	Value   string   `xml:"value,omitempty"`
}

// Boolean fields enable an entity to gather or provide an either-or choice
// between two options.
func Boolean(varName string, o ...FieldOption) Option {
	return func(data *Data) {
		f := field{
			Typ: "boolean",
			Var: varName,
		}
		getFieldOpts(&f, o...)
		data.children = append(data.children, f)
	}
}

func fieldOpt(typ, varName string, o ...FieldOption) func(data *Data) {
	return func(data *Data) {
		f := field{
			Typ: "typ",
			Var: varName,
		}
		getFieldOpts(&f, o...)
		data.children = append(data.children, f)
	}
}

// Fixed is intended for data description (e.g., human-readable text such as
// "section" headers) rather than data gathering or provision.
func Fixed(o ...FieldOption) Option {
	return fieldOpt("fixed", "")
}

// Hidden fields are not shown by the form-submitting entity, but instead are
// returned, generally unmodified, with the form.
func Hidden(varName string, o ...FieldOption) Option {
	return fieldOpt("hidden", varName)
}

// JIDMulti enables an entity to gather or provide multiple Jabber IDs.
func JIDMulti(varName string, o ...FieldOption) Option {
	return fieldOpt("jid-multi", varName)
}

// JID enables an entity to gather or provide a Jabber ID.
func JID(varName string, o ...FieldOption) Option {
	return fieldOpt("jid-single", varName)
}

// ListMulti enables an entity to gather or provide one or more entries from a
// list.
func ListMulti(varName string, o ...FieldOption) Option {
	return fieldOpt("list-multi", varName)
}

// ListSingle enables an entity to gather or provide a single entry from a list.
func ListSingle(varName string, o ...FieldOption) Option {
	return fieldOpt("list-single", varName)
}

// TextMulti enables an entity to gather or provide multiple lines of text.
func TextMulti(varName string, o ...FieldOption) Option {
	return fieldOpt("text-multi", varName)
}

// TextPrivate enables an entity to gather or provide a line of text that should
// be obscured in the submitting entities interface (eg. with multiple
// asterisks).
func TextPrivate(varName string, o ...FieldOption) Option {
	return fieldOpt("text-private", varName)
}

// TextSingle enables an entity to gather or provide a line of text.
func TextSingle(varName string, o ...FieldOption) Option {
	return fieldOpt("text-single", varName)
}
