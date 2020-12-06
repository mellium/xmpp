// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

// NS is the data forms namespace.
const NS = "jabber:x:data"

// Data represents a data form.
type Data struct {
	title struct {
		XMLName xml.Name `xml:"title"`
		Text    string   `xml:",chardata"`
	}
	typ      string
	children []xmlstream.Marshaler
}

// WriteXML implements xmlstream.WriterTo for Data.
func (d *Data) WriteXML(w xmlstream.TokenWriter) error {
	_, err := xmlstream.Copy(w, d.TokenReader())
	return err
}

// TokenReader implements xmlstream.Marshaler for Data.
func (d *Data) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Space: NS, Local: "x"}}
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "type"},
		Value: d.typ,
	})
	var child []xml.TokenReader
	// TODO: an "omit empty" Marshaler?
	if d.title.Text != "" {
		child = append(child, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(d.title.Text)),
			xml.StartElement{Name: xml.Name{Local: "title"}},
		))
	}
	for _, c := range d.children {
		child = append(child, c.TokenReader())
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(child...),
		start,
	)
}

// MarshalXML satisfies the xml.Marshaler interface for *Data.
func (d *Data) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	err := d.WriteXML(e)
	if err != nil {
		return err
	}
	return e.Flush()
}

type instructions struct {
	XMLName xml.Name `xml:"instructions"`
	Text    string   `xml:",chardata"`
}

func (i instructions) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.Token(xml.CharData(i.Text)),
		xml.StartElement{Name: xml.Name{Local: "instructions"}},
	)
}

// New builds a new data form from the provided options.
func New(o ...Field) *Data {
	form := &Data{typ: "form"}
	getOpts(form, o...)
	return form
}
