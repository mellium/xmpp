// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

func closePayload(sid string) xml.TokenReader {
	return xmlstream.Token(xml.StartElement{
		Name: xml.Name{Space: NS, Local: "close"},
		Attr: []xml.Attr{{
			Name:  xml.Name{Local: "sid"},
			Value: sid,
		}},
	})
}

type openIQ struct {
	stanza.IQ

	Open struct {
		BlockSize uint16 `xml:"block-size"`
		SID       string `xml:"sid"`
		Stanza    string `xml:"stanza,omitempty"`
	} `xml:"http://jabber.org/protocol/ibb open"`
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (iq openIQ) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (iq openIQ) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "open", Space: NS}}

	start.Attr = make([]xml.Attr, 0, 3)
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "block-size"},
		Value: strconv.FormatUint(uint64(iq.Open.BlockSize), 10),
	})
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "sid"},
		Value: iq.Open.SID,
	})
	if iq.Open.Stanza != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "stanza"},
			Value: iq.Open.Stanza,
		})
	}

	return stanza.WrapIQ(
		iq.IQ,
		xmlstream.Wrap(nil, start),
	)
}

type dataPayload struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/ibb data"`
	Seq     uint16   `xml:"seq,attr"`
	SID     string   `xml:"sid,attr"`
	data    []byte   `xml:",chardata"`
}

type dataIQ struct {
	stanza.IQ

	Data dataPayload `xml:"http://jabber.org/protocol/ibb data"`
}

type dataMessage struct {
	stanza.Message

	Data dataPayload `xml:"http://jabber.org/protocol/ibb data"`
}
