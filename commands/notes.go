// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands

//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -type=NoteType -linecomment

import (
	"encoding/xml"
	"fmt"

	"mellium.im/xmlstream"
)

// NoteType indicates the severity of a note.
// It should always be one of the pre-defined constants.
type NoteType int8

// MarshalXMLAttr satisfies xml.MarshalerAttr.
func (n NoteType) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	var err error
	if n < NoteInfo || n > NoteError {
		err = fmt.Errorf("invalid note type %s", n)
	}
	return xml.Attr{Name: name, Value: n.String()}, err
}

// UnmarshalXMLAttr satisfies xml.UnmarshalerAttr.
func (n *NoteType) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "info":
		*n = NoteInfo
	case "warn":
		*n = NoteWarn
	case "error":
		*n = NoteError
	default:
		*n = -1
		return fmt.Errorf("invalid note attribute %s", attr.Value)
	}
	return nil
}

// A list of possible NoteType's.
const (
	NoteInfo  NoteType = iota // info
	NoteWarn                  // warn
	NoteError                 // error
)

// Note provides information about the status of a command and may be returned
// as part of the response payload.
type Note struct {
	XMLName xml.Name `xml:"note"`
	Type    NoteType `xml:"type,attr"`
	Value   string   `xml:",cdata"`
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (n Note) TokenReader() xml.TokenReader {
	/* #nosec */
	attr, _ := n.Type.MarshalXMLAttr(xml.Name{Local: "type"})
	return xmlstream.Wrap(xmlstream.Token(xml.CharData(n.Value)), xml.StartElement{
		Name: xml.Name{Local: "note"},
		Attr: []xml.Attr{attr},
	})
}

// WriteXML satisfies the xmlstream.WriterTo interface.
func (n Note) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, n.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (n Note) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := n.WriteXML(e)
	return err
}
