// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

var spaceReplacer = strings.NewReplacer(
	"\r\n", " ",
	"\n\r", " ",
	"\n", " ",
	"\r", " ",
)

// Type is a form type. For more information see the constants defined in this
// package.
type Type string

const (
	// TypeForm indicates that the form-processing entity is asking the
	// form-submitting entity to complete a form.
	TypeForm Type = "form"

	// TypeSubmit indicates that the form-submitting entity is submitting data to
	// the form-processing entity.
	TypeSubmit Type = "submit"

	// TypeCancel indicates that the form-submitting entity has cancelled
	// submission of data to the form-processing entity.
	TypeCancel Type = "cancel"

	// TypeResult indicates that the form-processing entity is returning data
	// (e.g., search results) to the form-submitting entity, or the data is a
	// generic data set.
	TypeResult Type = "result"
)

// NS is the data forms namespace.
const NS = "jabber:x:data"

// Data represents a data form.
type Data struct {
	title        string
	instructions string
	typ          Type

	fields []field
	values map[string]interface{}
}

// Title returns the title of the form.
func (d *Data) Title() string {
	return d.title
}

// Instructions returns the instructions set on the form.
func (d *Data) Instructions() string {
	return d.instructions
}

// ForFields iterates over the fields of the form and calls a function for each
// one, passing it information about the field.
func (d *Data) ForFields(f func(FieldData)) {
	for _, field := range d.fields {
		f(FieldData{
			Type:     field.typ,
			Var:      field.varName,
			Label:    field.label,
			Desc:     field.desc,
			Required: field.required,
			Raw:      field.value,
		})
	}
}

// UnmarshalXML satisfies the xml.Unmarshaler interface for *Data.
func (d *Data) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			d.typ = Type(attr.Value)
			break
		}
	}

	d.values = make(map[string]interface{})

	for {
		tok, err := decoder.Token()
		if err != nil && err != io.EOF {
			return err
		}
		if tok == nil && err == io.EOF {
			return nil
		}

		switch t := tok.(type) {
		case xml.StartElement:
			start = t
		case xml.EndElement:
			return nil
		case xml.CharData:
			// Realistically we should never hit this in XMPP, but for the sake of
			// being able to parse forms from examples and the like skip over
			// chardata.
			continue
		default:
			// We shouldn't ever hit this, but just in case we do return an error.
			return errors.New("unexpected token type")
		}

		switch start.Name.Local {
		case "title":
			s := struct {
				XMLName xml.Name `xml:"title"`
				Inner   string   `xml:",chardata"`
			}{}
			err = decoder.DecodeElement(&s, &start)
			if err != nil {
				return err
			}
			d.title = s.Inner
		case "instructions":
			s := struct {
				XMLName xml.Name `xml:"instructions"`
				Inner   string   `xml:",chardata"`
			}{}
			err = decoder.DecodeElement(&s, &start)
			if err != nil {
				return err
			}
			if d.instructions == "" {
				d.instructions = s.Inner
			} else {
				d.instructions += "\n" + s.Inner
			}
		case "field":
			f := field{}
			err = decoder.DecodeElement(&f, &start)
			if err != nil {
				return err
			}
			if f.typ == "" {
				f.typ = TypeText
			}
			d.fields = append(d.fields, f)
		default:
			return fmt.Errorf("unexpected element %v", start.Name)
		}

		if err == io.EOF {
			break
		}
	}
	return decoder.Skip()
}

// Len returns the number of fields on the form.
func (d *Data) Len() int {
	if d == nil {
		return 0
	}
	return len(d.fields)
}

// Raw looks up the value parsed for a form field as it appeared in the XML.
func (d *Data) Raw(id string) (v []string, ok bool) {
	if d == nil {
		return nil, false
	}
	for _, field := range d.fields {
		if field.varName == id {
			return field.value, true
		}
	}
	return nil, false
}

// Get looks up the value submitted for a form field.
// If the value has not been set yet and no default value exists, ok will be
// false.
func (d *Data) Get(id string) (v interface{}, ok bool) {
	v, ok = d.values[id]
	if ok {
		return v, ok
	}
	fieldIDX := -1
	for i, field := range d.fields {
		if field.varName == id {
			fieldIDX = i
			break
		}
	}
	// No field with the given name found, so no default.
	if fieldIDX == -1 {
		return nil, false
	}

	// We found a field, so use its default.
	field := d.fields[fieldIDX]
	switch field.typ {
	case TypeFixed:
		// A submission of type fixed has no value.
		return "", false
	case TypeBoolean:
		for _, vv := range field.value {
			if vv == "false" || vv == "0" {
				return false, true
			}
			if vv == "true" || vv == "1" {
				return true, true
			}
		}
		return false, false
	case TypeText, TypeTextPrivate, TypeHidden, TypeList, "":
		if len(field.value) == 0 {
			return "", false
		}
		return field.value[0], true
	case TypeJID:
		for _, vv := range field.value {
			if j, err := jid.Parse(vv); err == nil {
				return j, true
			}
		}
		return jid.JID{}, false
	case TypeJIDMulti:
		var jids []jid.JID
		for _, vv := range field.value {
			if j, err := jid.Parse(vv); err == nil {
				jids = append(jids, j)
			}
		}
		return jids, len(jids) > 0
	case TypeTextMulti:
		b := &strings.Builder{}
		for i, vv := range field.value {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(vv)
		}
		return b.String(), len(field.value) > 0
	case TypeListMulti:
		var items []string
		items = append(items, field.value...)
		return items, len(items) > 0
	}
	return nil, false
}

// GetJID is like Get except that it asserts that the form submission is a JID.
// If the form submission was not a JID or is not set, ok will be false.
func (d *Data) GetJID(id string) (j jid.JID, ok bool) {
	v, ok := d.Get(id)
	if !ok {
		return j, ok
	}
	j, ok = v.(jid.JID)
	return j, ok
}

// GetString is like Get except that it asserts that the form submission is a
// string (regardless of whether it was a single line or multi-line submission).
// If the form submission was not a string or is not set, ok will be false.
func (d *Data) GetString(id string) (s string, ok bool) {
	v, ok := d.Get(id)
	if !ok {
		return s, ok
	}
	s, ok = v.(string)
	return s, ok
}

// GetStrings is like Get except that it asserts that the form submission is a
// slice of strings.
// If the form submission was not a string slice or is not set, ok will be
// false.
func (d *Data) GetStrings(id string) (s []string, ok bool) {
	v, ok := d.Get(id)
	if !ok {
		return s, ok
	}
	s, ok = v.([]string)
	return s, ok
}

// GetBool is like Get except that it asserts that the form submission is a
// bool.
// If the form submission was not a bool or is not set, ok will be false.
func (d *Data) GetBool(id string) (b, ok bool) {
	v, ok := d.Get(id)
	if !ok {
		return b, ok
	}
	b, ok = v.(bool)
	return b, ok
}

// GetJIDs is like Get except that it asserts that the form submission is a
// slice of JIDs.
// If the form submission was not a JID slice or is not set, ok will be false.
func (d *Data) GetJIDs(id string) (j []jid.JID, ok bool) {
	v, ok := d.Get(id)
	if !ok {
		return j, ok
	}
	j, ok = v.([]jid.JID)
	return j, ok
}

// Set sets the form field to the provided value.
// If the value is of the incorrect type for the form field an error is
// returned.
// If no form field with the given name exists, ok will be false.
// It is permitted to send back fields that did not exist in the original form
// so this is not an error, but most implementations will ignore them.
func (d *Data) Set(id string, v interface{}) (ok bool, err error) {
	var typ FieldType
	for _, field := range d.fields {
		if field.varName == id {
			typ = field.typ
			ok = true
			break
		}
	}
	switch typ {
	case TypeFixed:
		return false, fmt.Errorf("cannot set fixed field")
	case TypeBoolean:
		vv, isTyp := v.(bool)
		if !isTyp {
			return false, fmt.Errorf("expected %T, got %T", vv, v)
		}
	case TypeText, TypeTextPrivate, TypeHidden, TypeList, TypeTextMulti:
		vv, isTyp := v.(string)
		if !isTyp {
			return false, fmt.Errorf("expected %T, got %T", vv, v)
		}
	case TypeJID:
		vv, isTyp := v.(jid.JID)
		if !isTyp {
			return false, fmt.Errorf("expected %T, got %T", vv, v)
		}
	case TypeJIDMulti:
		vv, isTyp := v.([]jid.JID)
		if !isTyp {
			return false, fmt.Errorf("expected %T, got %T", vv, v)
		}
	case TypeListMulti:
		vv, isTyp := v.([]string)
		if !isTyp {
			return false, fmt.Errorf("expected %T, got %T", vv, v)
		}
	}
	d.values[id] = v
	return ok, err
}

// Submit returns a form that can be used to submit the original data.
// If a value has not been set for all required fields ok will be false.
func (d *Data) Submit() (submission xml.TokenReader, ok bool) {
	if d == nil {
		d = &Data{}
	}
	ok = true
	submissionData := New()
	submissionData.values = d.values
	submissionData.fields = d.fields
	submissionData.typ = TypeSubmit

	for _, f := range submissionData.fields {
		if f.required {
			_, isSet := submissionData.Get(f.varName)
			if !isSet {
				ok = false
			}
		}
	}
	return submissionData.TokenReader(), ok
}

// TokenReader implements xmlstream.Marshaler for Data.
func (d *Data) TokenReader() xml.TokenReader {
	var child []xml.TokenReader
	// Unwrap title (which cannot contain newlines)
	if d.title != "" {
		child = append(child, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(spaceReplacer.Replace(d.title))),
			xml.StartElement{Name: xml.Name{Local: "title"}},
		))
	}
	// Split instructions elements up one per line since the instructions element
	// cannot contain new lines (but there can be multiple of them).
	instructions := d.instructions
	for {
		idx := strings.IndexAny(instructions, "\r\n")
		line := instructions
		if idx != -1 {
			line = instructions[:idx]
			instructions = instructions[idx+1:]
			if line == "" {
				continue
			}
		}
		if line != "" {
			child = append(child, xmlstream.Wrap(
				xmlstream.Token(xml.CharData(line)),
				xml.StartElement{Name: xml.Name{Local: "instructions"}},
			))
		}
		if idx == -1 {
			break
		}
	}

	for _, f := range d.fields {
		// If we're type submit, skip unset fields and use the value from Get
		// instead of the raw field value (get returns defaults even if the field is
		// unset and may normalize some values).
		// Otherwise just append all fields exactly as they appear.
		if d.typ == TypeSubmit {
			if f.typ == TypeFixed {
				continue
			}
			vv, isSet := d.Get(f.varName)
			if !f.required && !isSet {
				continue
			}
			switch typed := vv.(type) {
			case []string:
				f.value = typed
			case string:
				if f.typ == TypeTextMulti {
					var lines []string
					for {
						idx := strings.IndexAny(typed, "\n\r")
						if idx == -1 {
							if len(typed) > 0 {
								lines = append(lines, typed)
								break
							}
						}
						lines = append(lines, typed[:idx])
						typed = typed[idx+1:]
					}
					f.value = lines
				} else {
					f.value = []string{typed}
				}
			case jid.JID:
				f.value = []string{typed.String()}
			case []jid.JID:
				f.value = make([]string, 0, len(typed))
				for _, j := range typed {
					f.value = append(f.value, j.String())
				}
			case bool:
				f.value = []string{strconv.FormatBool(typed)}
			}
		}
		child = append(child, f.TokenReader())
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(child...),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "x"},
			Attr: []xml.Attr{{
				Name:  xml.Name{Local: "type"},
				Value: string(d.typ),
			}},
		},
	)
}

// WriteXML implements xmlstream.WriterTo for Data.
func (d *Data) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface for *Data.
func (d *Data) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := d.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// New builds a new data form from the provided options.
func New(f ...Field) *Data {
	d := &Data{
		typ:    TypeForm,
		values: make(map[string]interface{}),
	}
	for _, field := range f {
		field(d)
	}
	return d
}

// Cancel returns a data form that can be used to cancel an interaction.
func Cancel(title, instructions string) *Data {
	d := New(Title(title), Instructions(instructions))
	d.typ = TypeCancel
	return d
}
