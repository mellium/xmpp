// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package upload

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// NS is the namespace used by this package.
const NS = "urn:xmpp:http:upload:0"

// File describes a file to be uploaded.
type File struct {
	Name string
	Size uint64
	Type string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (f File) TokenReader() xml.TokenReader {
	attr := []xml.Attr{
		{Name: xml.Name{Local: "filename"}, Value: f.Name},
		{Name: xml.Name{Local: "size"}, Value: strconv.FormatUint(f.Size, 10)},
	}
	if f.Type != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "content-type"}, Value: f.Type})
	}
	return xmlstream.Wrap(
		nil,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "request"},
			Attr: attr,
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (f File) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (f File) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := f.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (f *File) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	s := struct {
		XMLName xml.Name `xml:"urn:xmpp:http:upload:0 request"`
		Name    string   `xml:"filename,attr"`
		Size    uint64   `xml:"size,attr"`
		Type    string   `xml:"content-type,attr"`
	}{}
	err := d.DecodeElement(&s, &start)
	if err != nil {
		return err
	}
	f.Name = s.Name
	f.Size = s.Size
	f.Type = s.Type
	return nil
}

// Slot is a place where a file can be uploaded and later retrieved.
type Slot struct {
	PutURL *url.URL
	GetURL *url.URL

	// Header is the headers that will be set on put requests from the slot.
	// The only valid headers are "Authorization", "Cookie", and "Expires".
	// All other headers will be ignored.
	Header http.Header
}

func allowedHeader(name string) bool {
	return name == "Authorization" || name == "Cookie" || name == "Expires"
}

// Put returns a put request with the appropriate headers that can be used to
// upload a file to the slot.
func (s Slot) Put(ctx context.Context, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.PutURL.String(), body)
	if err != nil {
		return nil, err
	}
	headers := s.Header.Clone()
	for name := range headers {
		if !allowedHeader(http.CanonicalHeaderKey(name)) {
			headers.Del(name)
		}
	}
	req.Header = headers
	return req, nil
}

func marshalHeaders(h http.Header) xml.TokenReader {
	var headers []xml.TokenReader
	for name, vals := range h {
		name = http.CanonicalHeaderKey(name)
		if allowedHeader(name) {
			for _, val := range vals {
				headers = append(headers, xmlstream.Wrap(
					xmlstream.Token(xml.CharData(val)),
					xml.StartElement{
						Name: xml.Name{Local: "header"},
						Attr: []xml.Attr{{
							Name:  xml.Name{Local: "name"},
							Value: name,
						}},
					},
				))
			}
		}
	}
	return xmlstream.MultiReader(headers...)
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (s Slot) TokenReader() xml.TokenReader {
	var (
		putURL string
		getURL string
	)
	if s.PutURL != nil {
		putURL = s.PutURL.String()
	}
	if s.GetURL != nil {
		getURL = s.GetURL.String()
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Wrap(
				marshalHeaders(s.Header),
				xml.StartElement{
					Name: xml.Name{Local: "put"},
					Attr: []xml.Attr{{
						Name:  xml.Name{Local: "url"},
						Value: putURL,
					}},
				},
			),
			xmlstream.Wrap(
				nil,
				xml.StartElement{
					Name: xml.Name{Local: "get"},
					Attr: []xml.Attr{{
						Name:  xml.Name{Local: "url"},
						Value: getURL,
					}},
				},
			),
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "slot"}},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (s Slot) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, s.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (s Slot) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := s.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (s *Slot) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	v := struct {
		XMLName xml.Name `xml:"urn:xmpp:http:upload:0 slot"`
		Put     struct {
			URL    string `xml:"url,attr"`
			Header []struct {
				Name  string `xml:"name,attr"`
				Value string `xml:",cdata"`
			} `xml:"header"`
		} `xml:"put"`
		Get struct {
			URL string `xml:"url,attr"`
		} `xml:"get"`
	}{}
	err := d.DecodeElement(&v, &start)
	if err != nil {
		return err
	}
	if v.Get.URL != "" {
		s.GetURL, err = url.Parse(v.Get.URL)
		if err != nil {
			return err
		}
	}
	if v.Put.URL != "" {
		s.PutURL, err = url.Parse(v.Put.URL)
		if err != nil {
			return err
		}
	}
	for _, h := range v.Put.Header {
		if allowedHeader(http.CanonicalHeaderKey(h.Name)) {
			if s.Header == nil {
				s.Header = make(http.Header)
			}
			s.Header.Add(h.Name, h.Value)
		}
	}
	return nil
}

// GetSlot requests a URL where we can upload a file.
func GetSlot(ctx context.Context, f File, to jid.JID, s *xmpp.Session) (Slot, error) {
	return GetSlotIQ(ctx, f, stanza.IQ{
		To: to,
	}, s)
}

// GetSlotIQ is like GetSlot except that it lets you customize the IQ.
// Changing the type of the IQ has no effect.
func GetSlotIQ(ctx context.Context, f File, iq stanza.IQ, s *xmpp.Session) (Slot, error) {
	iq.Type = stanza.GetIQ
	var slot Slot
	err := s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "query"}},
	), iq, &slot)
	return slot, err
}
