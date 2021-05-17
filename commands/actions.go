// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -type=Actions -linecomment

// Actions represent the next steps that can be performed in multi-stage
// commands.
type Actions uint8

// A list of possible actions.
const (
	Prev     Actions = 1 << iota // prev
	Next                         // next
	Complete                     // complete

	// Execute is a bitmask that can be used to extract the default action.
	Execute = 0x38
)

// TokenReader satisfies the xmlstream.Marshaler interface.
func (a Actions) TokenReader() xml.TokenReader {
	var attr []xml.Attr
	switch execute := (a & Execute) >> 3; execute {
	case Prev, Next, Complete:
		attr = []xml.Attr{{Name: xml.Name{Local: "execute"}, Value: execute.String()}}
	default:
	}

	var inner []xml.TokenReader
	for i := Actions(1); i <= Complete; i <<= 1 {
		if a&i == 0 {
			continue
		}
		inner = append(inner, xmlstream.Wrap(nil, xml.StartElement{
			Name: xml.Name{Local: i.String()},
		}))
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(inner...),
		xml.StartElement{
			Name: xml.Name{Local: "actions"},
			Attr: attr,
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (a Actions) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, a.TokenReader())
}

// MarshalXML satisfies xml.Marshaler.
func (a Actions) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := a.WriteXML(e)
	return err
}

// UnmarshalXML satisfies xml.Unmarshaler.
func (a *Actions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var action Actions
	for _, attr := range start.Attr {
		if attr.Name.Local == "execute" {
			switch attr.Value {
			case "prev":
				action |= Prev << 3
			case "next":
				action |= Next << 3
			case "complete":
				action |= Complete << 3
			}
			break
		}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			break
		}
		switch start.Name.Local {
		case "prev":
			action |= Prev
		case "next":
			action |= Next
		case "complete":
			action |= Complete
		}
		err = d.Skip()
		if err != nil {
			return err
		}
	}
	*a = action
	return nil
}
