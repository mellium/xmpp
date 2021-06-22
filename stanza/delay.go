// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// Delay can be added to a stanza to indicate that stanza delivery was delayed.
// For example, when you joing a chat and request history, a delay might be
// added to indicate that the chat messages were sent in the past and are not
// live.
type Delay struct {
	From   jid.JID
	Stamp  time.Time
	Reason string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (d Delay) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(xmlstream.Token(xml.CharData(d.Reason)), xml.StartElement{
		Name: xml.Name{Space: NSDelay, Local: "delay"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "from"}, Value: d.From.String()},
			{Name: xml.Name{Local: "stamp"}, Value: d.Stamp.UTC().Format(time.RFC3339Nano)},
		},
	})
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (d Delay) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (d Delay) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := d.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (d *Delay) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	var foundFrom, foundStamp bool
	var err error
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "from":
			d.From, err = jid.Parse(attr.Value)
			if err != nil {
				return err
			}
			foundFrom = true
		case "stamp":
			d.Stamp, err = time.Parse(time.RFC3339Nano, attr.Value)
			if err != nil {
				return err
			}
			foundStamp = true
		}
		if foundFrom && foundStamp {
			break
		}
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case xml.EndElement:
		return nil
	case xml.CharData:
		d.Reason = string(t)
	case xml.StartElement:
		// There shouldn't be a start element in here, but tolerate unknown future
		// extensions and skip it if we find one.
		err = dec.Skip()
		if err != nil {
			return err
		}
	default:
	}
	return dec.Skip()
}
