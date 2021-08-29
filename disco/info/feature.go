// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package info contains service discovery features.
//
// These were separated out into a separate package to prevent import loops.
package info // import "mellium.im/xmpp/disco/info"

import (
	"encoding/xml"

	"mellium.im/xmlstream"
)

const (
	nsInfo = `http://jabber.org/protocol/disco#info`
)

// Feature represents a feature supported by an entity on the network.
type Feature struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info feature"`
	Var     string   `xml:"var,attr"`
}

// TokenReader implements xmlstream.Marshaler.
func (f Feature) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(nil, xml.StartElement{
		Name: xml.Name{Space: nsInfo, Local: "feature"},
		Attr: []xml.Attr{{
			Name:  xml.Name{Local: "var"},
			Value: f.Var,
		}},
	})
}

// WriteXML implements xmlstream.WriterTo.
func (f Feature) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (f Feature) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := f.WriteXML(e)
	return err
}

// FeatureIter is the interface implemented by types that implement disco
// features.
type FeatureIter interface {
	ForFeatures(node string, f func(Feature) error) error
}
