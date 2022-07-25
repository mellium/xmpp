// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package bin implements transfer of small bits-of-binary.
//
// Unlike Jingle File Transfer or In-Band Bytestreams which are designed for
// larger blobs of data, bits-of-binary does not require session negotiation.
package bin // import "mellium.im/xmpp/bin"

//go:generate go run ../internal/genfeature -receiver "Handler"

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"strconv"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// NS is the namespace implemented by this package.
const NS = "urn:xmpp:bob"

// Data contains a description of some binary data and, optionally, the data
// itself.
// It can be embedded in stanzas to transmit the data or a CID that can be used
// by the other side to fetch the data later.
// MaxAge is a hint for how long (rounded to the nearest second) the data should
// be cached.
// If MaxAge is set to the zero value, no cache hint is marshaled.
// To override this, set NoCache to true to send an explicit hint that the item
// should not be cached (this overrides MaxAge even if it is set to a non-zero
// value).
// Type is the MIME type as specified in RFC 2045 and is required for a
// non-empty data value.
type Data struct {
	CID     string
	MaxAge  time.Duration
	NoCache bool
	Type    string
	Data    []byte
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (d *Data) TokenReader() xml.TokenReader {
	var body xml.TokenReader
	if len(d.Data) > 0 {
		c := make(xml.CharData, base64.StdEncoding.EncodedLen(len(d.Data)))
		base64.StdEncoding.Encode(c, d.Data)
		body = xmlstream.Token(c)
	}
	attrs := make([]xml.Attr, 0, 3)
	if d.Type != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "type"}, Value: d.Type})
	}
	switch {
	case d.NoCache:
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "max-age"}, Value: "0"})
	case d.MaxAge > 0:
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "max-age"}, Value: strconv.FormatFloat(d.MaxAge.Seconds(), 'f', 0, 64)})
	}
	if d.CID != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "cid"}, Value: d.CID})
	}
	return xmlstream.Wrap(
		body,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "data"},
			Attr: attrs,
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (d *Data) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (d *Data) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := d.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (d *Data) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	v := struct {
		CID    string `xml:"cid,attr"`
		MaxAge *int64 `xml:"max-age,attr"`
		Type   string `xml:"type,attr"`
		Data   []byte `xml:",cdata"`
	}{}
	err := dec.DecodeElement(&v, &start)
	if err != nil {
		return err
	}
	d.CID = v.CID
	if v.MaxAge != nil {
		d.MaxAge = time.Duration(*v.MaxAge) * time.Second
	}
	d.NoCache = v.MaxAge != nil && *v.MaxAge == 0
	d.Type = v.Type
	d.Data = nil
	if l := len(v.Data); l > 0 {
		if decLen := base64.StdEncoding.DecodedLen(l); len(d.Data) < decLen {
			// TODO: consider re-using/extending the data buffer instead of
			// re-allocating every time.
			d.Data = make([]byte, decLen)
		}
		_, err = base64.StdEncoding.Decode(d.Data, v.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get requests the data associated with the given content ID URL.
func Get(ctx context.Context, s *xmpp.Session, to jid.JID, cid string) (*Data, error) {
	data := &Data{}
	err := s.UnmarshalIQ(
		ctx,
		stanza.IQ{
			Type: stanza.GetIQ,
			To:   to,
		}.Wrap((&Data{CID: cid}).TokenReader()),
		data,
	)
	return data, err
}

// Handler can be registered against a multiplexer using the Handle function to
// respond to requests for binary data.
// If the get function returns an error it is marshaled and sent in response to
// the request.
// If not, the returned data is sent.
// A nil Get function always returns an item-not-found error for all requests.
type Handler struct {
	Get func(cid string) (*Data, error)
}

// HandleIQ satisfies mux.IQHandler.
func (h Handler) HandleIQ(iq stanza.IQ, r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	data := &Data{}
	err := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), r)).Decode(data)
	if err != nil {
		return err
	}
	outData, err := h.Get(data.CID)
	stanzaErr := stanza.Error{}
	switch ok := errors.As(err, &stanzaErr); {
	case ok:
		_, err = xmlstream.Copy(r, iq.Error(stanzaErr))
		return err
	case err == nil:
		_, err = xmlstream.Copy(r, iq.Result(outData.TokenReader()))
		return err
	}
	return err
}

// Handle returns an option that when registered against a multiplexer handles
// incoming requests for content IDs.
//
// See Handler for more information.
func Handle(h Handler) mux.Option {
	if h.Get == nil {
		h.Get = func(string) (*Data, error) {
			return nil, stanza.Error{
				Type:      stanza.Cancel,
				Condition: stanza.ItemNotFound,
			}
		}
	}

	return mux.IQ(stanza.GetIQ, xml.Name{Space: NS, Local: "data"}, h)
}
