// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history

import (
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/paging"
)

// Result is the metadata (not messages) returned from a MAM query.
type Result struct {
	XMLName  xml.Name
	Complete bool
	Unstable bool
	Set      paging.Set
}

// TokenReader implements xmlstream.Marshaler.
func (r *Result) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		r.Set.TokenReader(),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "fin"},
			Attr: []xml.Attr{{
				Name:  xml.Name{Local: "complete"},
				Value: strconv.FormatBool(r.Complete),
			}, {
				Name:  xml.Name{Local: "stable"},
				Value: strconv.FormatBool(!r.Unstable),
			}},
		},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (r *Result) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, r.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (r *Result) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := r.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (r *Result) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var foundComplete, foundStable bool
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "complete":
			r.Complete = attr.Value == "true"
			foundComplete = true
		case "stable":
			r.Unstable = attr.Value == "false"
			foundStable = true
		}
		if foundComplete && foundStable {
			break
		}
	}
	var set paging.Set
	tok, err := d.Token()
	if err != nil {
		return err
	}
	start, ok := tok.(xml.StartElement)
	if !ok {
		return nil
	}
	err = d.DecodeElement(&set, &start)
	if err != nil {
		return err
	}
	r.Set = set
	return d.Skip()
}
