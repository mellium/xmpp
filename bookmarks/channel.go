// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks

import (
	"bytes"
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// Channel represents a single chat room with various properties.
type Channel struct {
	JID        jid.JID
	Autojoin   bool
	Name       string
	Nick       string
	Password   string
	Extensions []byte
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (c Channel) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	if c.Nick != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(c.Nick)),
			xml.StartElement{
				Name: xml.Name{Local: "nick"},
			},
		))
	}
	if c.Password != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(c.Password)),
			xml.StartElement{
				Name: xml.Name{Local: "password"},
			},
		))
	}
	if len(c.Extensions) > 0 {
		payloads = append(payloads, xmlstream.Wrap(
			xml.NewDecoder(bytes.NewReader(c.Extensions)),
			xml.StartElement{
				Name: xml.Name{Local: "extensions"},
			},
		))
	}
	conferenceAttrs := []xml.Attr{{
		Name:  xml.Name{Local: "autojoin"},
		Value: strconv.FormatBool(c.Autojoin),
	}}
	if c.Name != "" {
		conferenceAttrs = append(conferenceAttrs, xml.Attr{
			Name:  xml.Name{Local: "name"},
			Value: c.Name,
		})
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{
			Name: xml.Name{Local: "conference", Space: NS},
			Attr: conferenceAttrs,
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (c Channel) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, c.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (c Channel) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := c.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (c *Channel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	data := struct {
		XMLName    xml.Name `xml:"urn:xmpp:bookmarks:1 conference"`
		Name       string   `xml:"name,attr"`
		Autojoin   bool     `xml:"autojoin,attr"`
		Nick       string   `xml:"nick"`
		Password   string   `xml:"password"`
		Extensions struct {
			Val []byte `xml:",innerxml"`
		} `xml:"extensions"`
	}{}
	err := d.DecodeElement(&data, &start)
	if err != nil {
		return err
	}

	c.Autojoin = data.Autojoin
	c.Name = data.Name
	c.Nick = data.Nick
	c.Password = data.Password
	c.Extensions = data.Extensions.Val
	return nil
}
