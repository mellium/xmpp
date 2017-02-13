// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package form

import (
	"encoding/xml"
)

// NS is the data forms namespace.
const NS = "jabber:x:data"

var (
	formName = xml.Name{Space: "jabber:x:data", Local: "x"}
)

// Data represents a data form.
type Data struct {
	title struct {
		XMLName xml.Name `xml:"title"`
		Text    string   `xml:",chardata"`
	}
	typ      string
	children []interface{}
}

// MarshalXML satisfies the xml.Marshaler interface for *Data.
func (d *Data) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	start = xml.StartElement{Name: formName}
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "type"},
		Value: d.typ,
	})
	if err = e.EncodeToken(start); err != nil {
		return
	}

	// Encode the title.
	if d.title.Text != "" {
		if err = e.Encode(d.title); err != nil {
			return
		}
	}

	for _, c := range d.children {
		if err = e.Encode(c); err != nil {
			return
		}
	}

	// Encode the end element of the form.
	if err = e.EncodeToken(start.End()); err != nil {
		return
	}
	return e.Flush()
}

type instructions struct {
	XMLName xml.Name `xml:"instructions"`
	Text    string   `xml:",chardata"`
}

// New builds a new data form from the provided options.
func New(o ...Option) *Data {
	form := &Data{typ: "form"}
	getOpts(form, o...)
	return form
}
