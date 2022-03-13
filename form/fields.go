// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// FieldType is the type of fields in a dataform. For more information see the
// constants defined in this package.
type FieldType string

const (
	// TypeBoolean enables an entity to gather or provide an either-or choice
	// between two options.
	TypeBoolean FieldType = "boolean"

	// TypeFixed is intended for data description (e.g., human-readable text such
	// as "section" headers) rather than data gathering or provision.
	TypeFixed FieldType = "fixed"

	// TypeHidden is for fields that are not shown to the form-submitting entity,
	// but instead are returned with the form.
	TypeHidden FieldType = "hidden"

	// TypeJIDMulti enables an entity to gather or provide multiple JIDs.
	TypeJIDMulti FieldType = "jid-multi"

	// TypeJID enables an entity to gather or provide a single JID.
	TypeJID FieldType = "jid-single"

	// TypeListMulti enables an entity to gather or provide one or more options
	// from among many.
	TypeListMulti FieldType = "list-multi"

	// TypeList enables an entity to gather or provide one option from among many.
	TypeList FieldType = "list-single"

	// TypeTextMulti enables an entity to gather or provide multiple lines of
	// text.
	TypeTextMulti FieldType = "text-multi"

	// TypeTextPrivate enables an entity to gather or provide a single line or
	// word of text, which shall be obscured in an interface (e.g., with multiple
	// instances of the asterisk character).
	TypeTextPrivate FieldType = "text-private"

	// TypeText enables an entity to gather or provide a single line or word of
	// text, which may be shown in an interface.
	TypeText FieldType = "text-single"
)

// FieldOpt is an option on a field with type List or ListMulti.
type FieldOpt struct {
	Label string `xml:"label,attr"`
	Value string `xml:"value"`
}

// FieldData represents values from a single field in a data form.
// The Var field can then be passed to the Get and Set functions on the form to
// modify the field.
type FieldData struct {
	Type     FieldType
	Var      string
	Label    string
	Desc     string
	Required bool

	// Raw is the value of the field as it came over the wire with no type
	// information.
	// Generally speaking, Get methods on form should be used along with the field
	// data's Var value to fetch fields and Raw should be ignored.
	// Raw is mostly provided to access fixed type fields that do not have a
	// variable name (and therefore cannot be referenced or set).
	Raw []string
}

type field struct {
	typ      FieldType
	varName  string
	label    string
	desc     string
	value    []string
	option   []FieldOpt
	required bool
}

func (f *field) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	s := struct {
		Type     FieldType  `xml:"type,attr"`
		Label    string     `xml:"label,attr"`
		Var      string     `xml:"var,attr"`
		Desc     string     `xml:"desc"`
		Required *string    `xml:"required"`
		Value    []string   `xml:"value"`
		Option   []FieldOpt `xml:"option"`
	}{}

	err := d.DecodeElement(&s, &start)
	f.typ = s.Type
	f.label = s.Label
	f.varName = s.Var
	f.desc = s.Desc
	f.required = s.Required != nil
	f.value = s.Value
	f.option = s.Option
	return err
}

func (f *field) TokenReader() xml.TokenReader {
	attr := []xml.Attr{{
		Name:  xml.Name{Local: "type"},
		Value: string(f.typ),
	}}
	if f.varName != "" {
		attr = append(attr, xml.Attr{
			Name:  xml.Name{Local: "var"},
			Value: f.varName,
		})
	}
	if f.label != "" {
		attr = append(attr, xml.Attr{
			Name:  xml.Name{Local: "label"},
			Value: f.label,
		})
	}
	var child []xml.TokenReader
	if f.desc != "" {
		child = append(child, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(f.desc)),
			xml.StartElement{Name: xml.Name{Local: "desc"}},
		))
	}
	if f.required {
		child = append(child, xmlstream.Wrap(
			nil,
			xml.StartElement{Name: xml.Name{Local: "required"}},
		))
	}
	var firstVal bool
	for _, val := range f.value {
		if val == "" {
			continue
		}
		// Some list types are only allowed to have a single value.
		if firstVal && f.typ != "list-multi" && f.typ != "jid-multi" && f.typ != "text-multi" {
			break
		}
		switch f.typ {
		case TypeBoolean:
			if val != "true" && val != "false" && val != "0" && val != "1" {
				continue
			}
		case TypeJID, TypeJIDMulti:
			_, err := jid.Parse(val)
			if err != nil {
				continue
			}
		}
		child = append(child, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(val)),
			xml.StartElement{Name: xml.Name{Local: "value"}},
		))
		firstVal = true
	}
	if f.typ == "list-single" || f.typ == "list-multi" {
		for _, opt := range f.option {
			child = append(child, xmlstream.Wrap(
				xmlstream.Wrap(
					xmlstream.Token(xml.CharData(opt.Value)),
					xml.StartElement{Name: xml.Name{Local: "value"}},
				),
				xml.StartElement{
					Name: xml.Name{Local: "option"},
					Attr: []xml.Attr{{Name: xml.Name{Local: "label"}, Value: opt.Label}},
				},
			))
		}
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(child...),
		xml.StartElement{
			Name: xml.Name{Local: "field"},
			Attr: attr,
		},
	)
}

func newField(typ FieldType, id string, o ...Option) func(data *Data) {
	return func(data *Data) {
		f := field{
			typ:     typ,
			varName: id,
		}
		getFieldOpts(&f, o...)
		data.fields = append(data.fields, f)
	}
}

// Boolean fields enable an entity to gather or provide an either-or choice
// between two options.
func Boolean(id string, o ...Option) Field {
	return newField(TypeBoolean, id, o...)
}

// Fixed is intended for data description (e.g., human-readable text such as
// "section" headers) rather than data gathering or provision.
func Fixed(o ...Option) Field {
	return newField(TypeFixed, "", o...)
}

// Hidden fields are not shown by the form-submitting entity, but instead are
// returned, generally unmodified, with the form.
func Hidden(id string, o ...Option) Field {
	return newField(TypeHidden, id, o...)
}

// JIDMulti enables an entity to gather or provide multiple Jabber IDs.
func JIDMulti(id string, o ...Option) Field {
	return newField(TypeJIDMulti, id, o...)
}

// JID enables an entity to gather or provide a Jabber ID.
func JID(id string, o ...Option) Field {
	return newField(TypeJID, id, o...)
}

// ListMulti enables an entity to gather or provide one or more entries from a
// list.
func ListMulti(id string, o ...Option) Field {
	return newField(TypeListMulti, id, o...)
}

// List enables an entity to gather or provide a single entry from a list.
func List(id string, o ...Option) Field {
	return newField(TypeList, id, o...)
}

// TextMulti enables an entity to gather or provide multiple lines of text.
func TextMulti(id string, o ...Option) Field {
	return newField(TypeTextMulti, id, o...)
}

// TextPrivate enables an entity to gather or provide a line of text that should
// be obscured in the submitting entities interface (eg. with multiple
// asterisks).
func TextPrivate(id string, o ...Option) Field {
	return newField(TypeTextPrivate, id, o...)
}

// Text enables an entity to gather or provide a line of text.
func Text(id string, o ...Option) Field {
	return newField(TypeText, id, o...)
}
