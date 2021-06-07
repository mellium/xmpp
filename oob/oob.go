// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -vars=Feature:NS,FeatureIQ:NS

// Package oob implements XEP-0066: Out of Band Data.
package oob // import "mellium.im/xmpp/oob"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

// OOB namespaces provided as a convenience.
const (
	NS      = `jabber:x:oob`
	NSQuery = `jabber:iq:oob`
)

// IQ represents an OOB data query; for instance:
//
//     <iq type='set'
//         from='feste@example.net/asegasd'
//         to='malvolio@jabber.org/apkjase'
//         id='asiepjg'>
//       <query xmlns='jabber:iq:oob'>
//         <url>https://xmpp.org/images/promo/xmpp_server_guide_2017.pdf</url>
//         <desc>XMPP Server Setup Guide 2017</desc>
//       </query>
//     </iq>
type IQ struct {
	stanza.IQ
	Query Query
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (iq IQ) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (iq IQ) TokenReader() xml.TokenReader {
	return iq.IQ.Wrap(iq.Query.TokenReader())
}

// Query represents an OOB data node that might be placed in an IQ stanza.
type Query struct {
	XMLName xml.Name `xml:"jabber:iq:oob query"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (q Query) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (q Query) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "query", Space: NSQuery}}
	return xmlstream.Wrap(getPayload(q.URL, q.Desc), start)
}

// Data represents an OOB data node that might be placed in a message or
// presence stanza.
type Data struct {
	XMLName xml.Name `xml:"jabber:x:oob x"`
	URL     string   `xml:"url"`
	Desc    string   `xml:"desc,omitempty"`
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (d Data) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, d.TokenReader())
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (d Data) TokenReader() xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Local: "x", Space: NS}}
	return xmlstream.Wrap(getPayload(d.URL, d.Desc), start)
}

func getPayload(url, desc string) xml.TokenReader {
	// Create the payload and add the URL element.
	payload := xmlstream.Wrap(xmlstream.Token(xml.CharData(url)), xml.StartElement{
		Name: xml.Name{Local: "url"},
	})

	// Escape and append <desc> element if it's not empty.
	if desc != "" {
		payload = xmlstream.MultiReader(payload, xmlstream.Wrap(xmlstream.Token(xml.CharData(desc)), xml.StartElement{
			Name: xml.Name{Local: "desc"},
		}))
	}

	return payload
}
