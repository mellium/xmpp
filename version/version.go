// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature

// Package version queries a remote entity for software version info.
package version // import "mellium.im/xmpp/version"

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	// NS is the XML namespace used by software version queries.
	// It is provided as a convenience.
	NS = "jabber:iq:version"
)

// Query is the payload of a software version query or response.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:version query"`
	Name    string   `xml:"name,omitempty"`
	Version string   `xml:"version,omitempty"`
	OS      string   `xml:"os,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (q Query) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	if q.Name != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(q.Name)),
			xml.StartElement{Name: xml.Name{Local: "name"}},
		))
	}
	if q.Version != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(q.Version)),
			xml.StartElement{Name: xml.Name{Local: "version"}},
		))
	}
	if q.OS != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(q.OS)),
			xml.StartElement{Name: xml.Name{Local: "os"}},
		))
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "query"}},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (q Query) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}

// Get requests the software version of the provided entity.
// It blocks until a response is received.
func Get(ctx context.Context, s *xmpp.Session, to jid.JID) (Query, error) {
	return GetIQ(ctx, stanza.IQ{To: to}, s)
}

// GetIQ is like Get but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func GetIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) (Query, error) {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}

	query := Query{}
	err := s.UnmarshalIQ(ctx, iq.Wrap(query.TokenReader()), &query)
	return query, err
}
