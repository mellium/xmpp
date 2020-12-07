// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
)

// NS is the data forms namespace.
const NS = "jabber:x:data"

// Type is the type of the form. It should normally be one of the constants
// defined in this package.
type Type string

const (
	// TypeForm is used when the form-processing entity is asking the
	// form-submitting entity to complete a form.
	TypeForm = "form"

	// TypeSubmit is used when the form-submitting entity is submitting data to
	// the form-processing entity
	TypeSubmit = "submit"

	// TypeCancel is used when the form-submitting entity has cancelled submission
	// of data to the form-processing entity.
	TypeCancel = "cancel"

	// TypeResult is used when the form-processing entity is returning data (e.g.,
	// search results) to the form-submitting entity, or the data is a generic
	// data set.
	TypeResult = "result"
)

// Data represents a data form.
// If Type is unset, TypeForm will be used by default.
type Data struct {
	Title        string
	Instructions string
	Type         Type
	Fields       []Field
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (d Data) TokenReader() xml.TokenReader {
	typ := d.Type
	if typ == "" {
		typ = TypeForm
	}
	inner := []xml.TokenReader{
		xmlstream.Wrap(
			xmlstream.Token(xml.CharData(d.Title)),
			xml.StartElement{Name: xml.Name{Local: "title"}},
		),
	}
	inst := d.Instructions
	for {
		idx := strings.IndexAny(inst, "\n\r")
		if idx == -1 {
			break
		}
		line := inst[:idx]
		inst = inst[idx+1:]
		if line == "" {
			continue
		}
		inner = append(inner, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(line)),
			xml.StartElement{Name: xml.Name{Local: "instructions"}},
		))
	}
	for _, field := range d.Fields {
		inner = append(inner, field.TokenReader())
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(inner...),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "x"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "type"}, Value: string(typ)}},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (d Data) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (d Data) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := d.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}
