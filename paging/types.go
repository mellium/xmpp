// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package paging

import (
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
)

// RequestCount can be added to a query to request the count of elements without
// returning any actual items.
type RequestCount struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
}

// TokenReader implements xmlstream.Marshaler.
func (req *RequestCount) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Token(xml.CharData("0")),
			xml.StartElement{Name: xml.Name{Local: "max"}},
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "set"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (req *RequestCount) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, req.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (req *RequestCount) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := req.WriteXML(e)
	return err
}

// RequestNext can be added to a query to request the first page or to page
// forward.
type RequestNext struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
	Max     uint64   `xml:"max,omitempty"`
	After   string   `xml:"after,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (req *RequestNext) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	if req.Max > 0 {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(strconv.FormatUint(req.Max, 10))),
			xml.StartElement{Name: xml.Name{Local: "max"}},
		))
	}
	if req.After != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(req.After)),
			xml.StartElement{Name: xml.Name{Local: "after"}},
		))
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "set"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (req *RequestNext) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, req.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (req *RequestNext) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := req.WriteXML(e)
	return err
}

// RequestPrev can be added to a query to request the last page or to page
// backward.
type RequestPrev struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
	Max     uint64   `xml:"max,omitempty"`
	Before  string   `xml:"before"`
}

// TokenReader implements xmlstream.Marshaler.
func (req *RequestPrev) TokenReader() xml.TokenReader {
	payloads := []xml.TokenReader{xmlstream.Wrap(
		xmlstream.Token(xml.CharData(req.Before)),
		xml.StartElement{Name: xml.Name{Local: "before"}},
	)}
	if req.Max > 0 {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(strconv.FormatUint(req.Max, 10))),
			xml.StartElement{Name: xml.Name{Local: "max"}},
		))
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "set"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (req *RequestPrev) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, req.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (req *RequestPrev) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := req.WriteXML(e)
	return err
}

// RequestIndex can be added to a query to skip to a specific page.
// It is not always supported.
type RequestIndex struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
	Max     uint64   `xml:"max"`
	Index   uint64   `xml:"index"`
}

// TokenReader implements xmlstream.Marshaler.
func (req *RequestIndex) TokenReader() xml.TokenReader {
	payloads := []xml.TokenReader{xmlstream.Wrap(
		xmlstream.Token(xml.CharData(strconv.FormatUint(req.Index, 10))),
		xml.StartElement{Name: xml.Name{Local: "index"}},
	), xmlstream.Wrap(
		xmlstream.Token(xml.CharData(strconv.FormatUint(req.Max, 10))),
		xml.StartElement{Name: xml.Name{Local: "max"}},
	)}
	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "set"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (req *RequestIndex) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, req.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (req *RequestIndex) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := req.WriteXML(e)
	return err
}

// Set describes a page from a returned result set.
type Set struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/rsm set"`
	First   struct {
		ID    string  `xml:",cdata"`
		Index *uint64 `xml:"index,attr,omitempty"`
	} `xml:"first"`
	Last  string  `xml:"last"`
	Count *uint64 `xml:"count,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (s *Set) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	start := xml.StartElement{Name: xml.Name{Local: "first"}}
	if s.First.Index != nil {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "index"},
			Value: strconv.FormatUint(*s.First.Index, 10),
		})
	}
	payloads = append(payloads, xmlstream.Wrap(
		xmlstream.Token(xml.CharData(s.First.ID)),
		start,
	))
	payloads = append(payloads, xmlstream.Wrap(
		xmlstream.Token(xml.CharData(s.Last)),
		xml.StartElement{Name: xml.Name{Local: "last"}},
	))
	if s.Count != nil {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(strconv.FormatUint(*s.Count, 10))),
			xml.StartElement{Name: xml.Name{Local: "count"}},
		))
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "set"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (s *Set) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, s.TokenReader())
}

// MarshalXML satisfies the xml.Marshaler interface.
func (s *Set) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := s.WriteXML(e)
	return err
}
