// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package file contains shared functionality between various file upload
// mechanisms.
package file

import (
	"encoding/xml"
	"strconv"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/crypto"
)

const (
	NSMeta = `urn:xmpp:file:metadata:0`
)

// TODO: the "desc" element is not specified on meta. We need a better way to
// handle xml:lang first.
// TODO: the "thumbnail" element is not specified, we need to implement
// XEP-02654: File Transfer Thumbnails first.

// Meta is an element that contains metadata about a file.
type Meta struct {
	MediaType string
	Name      string
	Date      time.Time
	Size      uint64
	Hash      crypto.HashOutput
	Width     uint64
	Height    uint64
	Length    uint64
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (m *Meta) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(m.MediaType)),
				xml.StartElement{
					Name: xml.Name{Local: "media-type"},
				},
			),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(m.Name)),
				xml.StartElement{
					Name: xml.Name{Local: "name"},
				},
			),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(m.Date.Format(time.RFC3339))),
				xml.StartElement{
					Name: xml.Name{Local: "date"},
				},
			),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(strconv.FormatUint(m.Size, 10))),
				xml.StartElement{
					Name: xml.Name{Local: "size"},
				},
			),
			m.Hash.TokenReader(),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(strconv.FormatUint(m.Width, 10))),
				xml.StartElement{
					Name: xml.Name{Local: "width"},
				},
			),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(strconv.FormatUint(m.Height, 10))),
				xml.StartElement{
					Name: xml.Name{Local: "height"},
				},
			),
			xmlstream.Wrap(
				xmlstream.Token(xml.CharData(strconv.FormatUint(m.Length, 10))),
				xml.StartElement{
					Name: xml.Name{Local: "length"},
				},
			),
		),
		xml.StartElement{
			Name: xml.Name{Space: NSMeta, Local: "file"},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (m *Meta) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, m.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (m *Meta) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := m.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (m *Meta) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	in := &struct {
		XMLName   xml.Name          `xml:"urn:xmpp:file:metadata:0 file"`
		MediaType string            `xml:"media-type"`
		Name      string            `xml:"name"`
		Date      time.Time         `xml:"date"`
		Size      uint64            `xml:"size"`
		Hash      crypto.HashOutput `xml:"hash"`
		Width     uint64            `xml:"width"`
		Height    uint64            `xml:"height"`
		Length    uint64            `xml:"length"`
	}{}
	err := d.DecodeElement(&in, &start)
	if err != nil {
		return err
	}
	m.MediaType = in.MediaType
	m.Name = in.Name
	m.Date = in.Date
	m.Size = in.Size
	m.Hash = in.Hash
	m.Width = in.Width
	m.Height = in.Height
	m.Length = in.Length
	return nil
}
