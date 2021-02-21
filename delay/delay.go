// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package delay implements delayed delivery of stanzas.
package delay // import "mellium.im/xmpp/delay"

import (
	"encoding/xml"
	"fmt"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/xtime"
)

// NS is the namespace used by this package.
const NS = "urn:xmpp:delay"

// Delay is a type that can be added to stanzas to indicate that they have been
// delivered with a delay.
type Delay struct {
	XMLName xml.Name  `xml:"urn:xmpp:delay delay"`
	From    jid.JID   `xml:"from,attr,omitempty"`
	Time    time.Time `xml:"stamp,attr"`
	Reason  string    `xml:",chardata"`
}

// TokenReader implements xmlstream.Marshaler.
func (d Delay) TokenReader() xml.TokenReader {
	timeAttr, err := xtime.Time{Time: d.Time}.MarshalXMLAttr(xml.Name{Local: "stamp"})
	if err != nil {
		panic(fmt.Errorf("delay: unreachable error reached while marshaling time: %w", err))
	}
	start := xml.StartElement{
		Name: xml.Name{Space: NS, Local: "delay"},
		Attr: []xml.Attr{timeAttr},
	}

	if !d.From.Equal(jid.JID{}) {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "from"},
			Value: d.From.String(),
		})
	}

	if d.Reason != "" {
		return xmlstream.Wrap(xmlstream.Token(xml.CharData(d.Reason)), start)
	}
	return xmlstream.Wrap(nil, start)
}

// WriteXML implements xmlstream.WriterTo.
func (d Delay) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (d Delay) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := d.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (d *Delay) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	var err error
	var foundStamp, foundFrom bool
	for _, attr := range start.Attr {
		if attr.Name.Space != "" && attr.Name.Space != NS {
			continue
		}
		switch attr.Name.Local {
		case "stamp":
			foundStamp = true
			var xt xtime.Time
			err = (&xt).UnmarshalXMLAttr(attr)
			d.Time = xt.Time
		case "from":
			foundFrom = true
			err = (&d.From).UnmarshalXMLAttr(attr)
		}
		if err != nil {
			return err
		}
		if foundStamp && foundFrom {
			break
		}
	}
	tok, err := decoder.Token()
	if err != nil {
		return err
	}
	switch data := tok.(type) {
	case xml.CharData:
		d.Reason = string(data)
	case xml.EndElement:
		return nil
	}
	return decoder.Skip()
}

// TODO: replace when #113 is ready.
func isStanza(name xml.Name) bool {
	return (name.Local == "iq" || name.Local == "message" || name.Local == "presence") &&
		(name.Space == ns.Client || name.Space == ns.Server)
}

// Stanza inserts a delay into any stanza read through the stream.
func Stanza(d Delay) xmlstream.Transformer {
	return xmlstream.InsertFunc(func(start xml.StartElement, level uint64, w xmlstream.TokenWriter) error {
		if !isStanza(start.Name) || level != 1 {
			return nil
		}

		_, err := xmlstream.Copy(w, d.TokenReader())
		return err
	})
}

// Insert adds a delay into any element read through the transformer at the
// current nesting level.
func Insert(d Delay) xmlstream.Transformer {
	return xmlstream.InsertFunc(func(start xml.StartElement, level uint64, w xmlstream.TokenWriter) error {
		if level != 1 {
			return nil
		}

		_, err := xmlstream.Copy(w, d.TokenReader())
		return err
	})
}
