// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package pubsub

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/stanza"
)

// GetConfig fetches the configurable options for the given node.
func GetConfig(ctx context.Context, s *xmpp.Session, node string) (*form.Data, error) {
	return GetConfigIQ(ctx, s, stanza.IQ{}, node)
}

// GetConfigIQ is like GetConfig except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func GetConfigIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node string) (*form.Data, error) {
	return getConfig(ctx, s, iq, node, false)
}

// GetDefaultConfig fetches the configurable options for the given node.
func GetDefaultConfig(ctx context.Context, s *xmpp.Session) (*form.Data, error) {
	return GetDefaultConfigIQ(ctx, s, stanza.IQ{})
}

// GetDefaultConfigIQ is like GetDefaultConfig except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func GetDefaultConfigIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ) (*form.Data, error) {
	return getConfig(ctx, s, iq, "", true)
}

func getConfig(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node string, def bool) (*form.Data, error) {
	iq.Type = stanza.GetIQ
	var resp struct {
		XMLName   xml.Name `xml:"http://jabber.org/protocol/pubsub#owner pubsub"`
		Configure struct {
			XMLName xml.Name   `xml:"configure"`
			Data    *form.Data `xml:"jabber:x:data x"`
		} `xml:"configure"`
		Default struct {
			XMLName xml.Name   `xml:"default"`
			Data    *form.Data `xml:"jabber:x:data x"`
		} `xml:"default"`
	}

	start := xml.StartElement{Name: xml.Name{Local: "default"}}
	if !def {
		start = xml.StartElement{Name: xml.Name{Local: "configure"}, Attr: []xml.Attr{{Name: xml.Name{Local: "node"}, Value: node}}}
	}

	err := s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			nil,
			start,
		),
		xml.StartElement{Name: xml.Name{Space: NSOwner, Local: "pubsub"}},
	), iq, &resp)

	if def {
		return resp.Default.Data, err
	}
	return resp.Configure.Data, err
}

// SetConfig submits the provided dataform to the server for the given node.
func SetConfig(ctx context.Context, s *xmpp.Session, node string, cfg *form.Data) error {
	return SetConfigIQ(ctx, s, stanza.IQ{}, node, cfg)
}

// SetConfigIQ is like SetConfig except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func SetConfigIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node string, cfg *form.Data) error {
	iq.Type = stanza.SetIQ
	data, _ := cfg.Submit()
	return s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			data,
			xml.StartElement{Name: xml.Name{Local: "configure"}, Attr: []xml.Attr{{Name: xml.Name{Local: "node"}, Value: node}}},
		),
		xml.StartElement{Name: xml.Name{Space: NSOwner, Local: "pubsub"}},
	), iq, nil)
}
