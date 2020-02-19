// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xtime implements time related XMPP functionality.
//
// In particular, this package implements XEP-0202: Entity Time and XEP-0082:
// XMPP Date and Time Profiles.
package xtime // import "mellium.im/xmpp/xtime"

import (
	"context"
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const (
	// NS is the XML namespace used by XMPP entity time requests.
	// It is provided as a convenience.
	NS = "urn:xmpp:time"

	// LegacyDateTime implements the legacy profile mentioned in XEP-0082.
	//
	// Unless you are implementing an older XEP that specifically calls for this
	// format, time.RFC3339 should be used instead.
	LegacyDateTime = "20060102T15:04:05"
)

const tzd = "Z07:00"

// Time is like a time.Time but it can be marshaled as an XEP-0202 time payload.
type Time time.Time

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (t Time) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, t.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (t Time) TokenReader() xml.TokenReader {
	tt := time.Time(t)
	tzo := tt.Format(tzd)
	utcTime := tt.UTC().Format(time.RFC3339)

	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Wrap(xmlstream.Token(xml.CharData(tzo)), xml.StartElement{Name: xml.Name{Local: "tzo"}}),
			xmlstream.Wrap(xmlstream.Token(xml.CharData(utcTime)), xml.StartElement{Name: xml.Name{Local: "utc"}}),
		),
		xml.StartElement{Name: xml.Name{Local: "time", Space: NS}},
	)
}

// MarshalXML implements xml.Marshaler.
func (t Time) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := t.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	data := struct {
		XMLName xml.Name `xml:"urn:xmpp:time time"`

		Timezone string `xml:"tzo"`
		UTC      string `xml:"utc"`
	}{}
	err := d.DecodeElement(&data, &start)
	if err != nil {
		return err
	}

	zone, err := time.Parse(tzd, data.Timezone)
	if err != nil {
		return err
	}
	utcTime, err := time.Parse(time.RFC3339, data.UTC)
	if err != nil {
		return err
	}
	*t = Time(utcTime.In(zone.Location()))
	return nil
}

// Get sends a request to the provided JID asking for its time.
func Get(ctx context.Context, s *xmpp.Session, to jid.JID) (time.Time, error) {
	result, err := s.SendIQ(ctx, stanza.WrapIQ(stanza.IQ{
		Type: stanza.GetIQ,
		To:   to,
	}, xmlstream.Wrap(nil, xml.StartElement{Name: xml.Name{Local: "time", Space: NS}})))
	var t time.Time
	if err != nil {
		return t, err
	}
	d := xml.NewTokenDecoder(result)
	data := struct {
		stanza.IQ
		Time Time
	}{}
	err = d.Decode(&data)
	if err != nil {
		return t, err
	}
	return time.Time(data.Time), nil
}

// Handle returns an option that registers a Handler for entity time requests.
func Handle(h Handler) mux.Option {
	return mux.IQ(stanza.GetIQ, xml.Name{Local: "time", Space: NS}, h)
}

// Handler responds to requests for our time.
// If timeFunc is nil, time.Now is used.
type Handler struct {
	TimeFunc func() time.Time
}

// HandleIQ responds to entity time requests.
func (h Handler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if iq.Type != stanza.GetIQ || start.Name.Local != "time" || start.Name.Space != NS {
		return nil
	}

	var tt Time
	if h.TimeFunc == nil {
		tt = Time(time.Now())
	} else {
		tt = Time(h.TimeFunc())
	}

	_, err := xmlstream.Copy(t, iq.Result(tt.TokenReader()))
	return err
}
