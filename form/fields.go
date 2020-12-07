// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

import (
	"encoding/xml"
	"strconv"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

var newlineReplacer = strings.NewReplacer(
	"\r\n", " ",
	"\r", " ",
	"\n", " ",
)

// Field is the interface implemented by types that can be embedded in data
// forms as fields.
type Field interface {
	xml.Marshaler
	xmlstream.Marshaler
	xmlstream.WriterTo
	tokenReader(string, ...xml.TokenReader) xml.TokenReader
}

// Common fulfills part of the Field interface and holds common struct fields
// used by data form field types.
type Common struct {
	Var      string
	Label    string
	Desc     string
	Required bool
}

func (c Common) tokenReader(typ string, value ...xml.TokenReader) xml.TokenReader {
	start := xml.StartElement{
		Name: xml.Name{Local: "field"},
		Attr: []xml.Attr{},
	}
	if c.Var != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "var"}, Value: c.Var})
	}
	if c.Label != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "label"}, Value: c.Label})
	}

	var inner []xml.TokenReader
	if c.Desc != "" {
		inner = append(inner, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(c.Desc)),
			xml.StartElement{Name: xml.Name{Local: "desc"}},
		))
	}
	if c.Required {
		inner = append(inner, xmlstream.Wrap(
			nil,
			xml.StartElement{Name: xml.Name{Local: "required"}},
		))
	}
	for _, v := range value {
		inner = append(inner, xmlstream.Wrap(v, xml.StartElement{Name: xml.Name{Local: "value"}}))
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(inner...),
		start,
	)
}

// Boolean fields enable an entity to gather or provide an either-or choice
// between two options.
type Boolean struct {
	Common
	Value bool
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (b Boolean) TokenReader() xml.TokenReader {
	return b.tokenReader("boolean", xmlstream.Token(xml.CharData(strconv.FormatBool(b.Value))))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (b Boolean) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, b.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (b Boolean) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := b.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

func lineValues(c Common, typ, s string) xml.TokenReader {
	var fields []xml.TokenReader
	for {
		idx := strings.IndexAny(s, "\n\r")
		if idx == -1 {
			break
		}
		line := s[:idx]
		s = s[idx+1:]
		if line == "" {
			continue
		}
		fields = append(fields, c.tokenReader(typ, xmlstream.Token(xml.CharData(line))))
	}
	return xmlstream.MultiReader(fields...)
}

// Fixed is intended for data description (e.g., human-readable text such as
// "section" headers) rather than data gathering or provision.
type Fixed struct {
	Common
	Value string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (f Fixed) TokenReader() xml.TokenReader {
	return lineValues(f.Common, "fixed", f.Value)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (f Fixed) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (f Fixed) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := f.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// Hidden fields are not shown by the form-submitting entity, but instead are
// returned, generally unmodified, with the form.
type Hidden struct {
	Common
	Value string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (h Hidden) TokenReader() xml.TokenReader {
	return h.tokenReader("hidden", xmlstream.Token(xml.CharData(h.Value)))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (h Hidden) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, h.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (h Hidden) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := h.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// JIDMulti enables an entity to gather or provide multiple Jabber IDs.
type JIDMulti struct {
	Common
	Value []jid.JID
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (j JIDMulti) TokenReader() xml.TokenReader {
	var values []xml.TokenReader
	for _, v := range j.Value {
		values = append(values, xmlstream.Token(xml.CharData(v.String())))
	}
	return j.tokenReader("jid-multi", values...)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (j JIDMulti) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, j.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (j JIDMulti) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := j.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// JID enables an entity to gather or provide a single Jabber ID.
type JID struct {
	Common
	Value jid.JID
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (j JID) TokenReader() xml.TokenReader {
	return j.tokenReader("jid-single", xmlstream.Token(xml.CharData(j.Value.String())))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (j JID) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, j.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (j JID) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := j.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// ListItem is an option for the List and ListMulti types.
type ListItem struct {
	XMLName xml.Name `xml:"option"`
	Label   string   `xml:"label,attr"`
	Value   string   `xml:"value"`
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (l ListItem) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Token(xml.CharData(l.Value)),
			xml.StartElement{Name: xml.Name{Local: "value"}},
		),
		xml.StartElement{
			Name: xml.Name{Local: "option"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "label"}, Value: l.Label}},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (l ListItem) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, l.TokenReader())
}

// List enables an entity to gather or provide one or more entries from a list.
type List struct {
	Common
	Multi bool
	Value []ListItem
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (l List) TokenReader() xml.TokenReader {
	var values []xml.TokenReader
	for _, item := range l.Value {
		values = append(values, item.TokenReader())
	}
	if l.Multi {
		return l.tokenReader("list-multi", values...)
	}
	return l.tokenReader("list-single", values...)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (l List) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, l.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (l List) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := l.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// TextMulti enables an entity to gather or provide multiple lines of text.
type TextMulti struct {
	Common
	Value string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (t TextMulti) TokenReader() xml.TokenReader {
	return lineValues(t.Common, "text-multi", t.Value)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (t TextMulti) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, t.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (t TextMulti) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := t.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

// Text enables an entity to gather or provide a line of text.
// If Private is true the text should be obscured (eg. with asterisks) and may
// be sensitive.
type Text struct {
	Common
	Private bool
	Value   string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
// If the value contains any newlines, they will be replaced with spaces.
func (t Text) TokenReader() xml.TokenReader {
	typ := "text-single"
	if t.Private {
		typ = "text-private"
	}
	val := newlineReplacer.Replace(t.Value)
	return t.tokenReader(typ, xmlstream.Token(xml.CharData(val)))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (t Text) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, t.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (t Text) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := t.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}
