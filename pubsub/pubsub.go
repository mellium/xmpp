// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package pubsub

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/stanza"
)

type publishResponse struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Publish struct {
		Item struct {
			ID string `xml:"id,attr"`
		} `xml:"item"`
	} `xml:"publish"`
}

// Publish copies the first element from the provided token reader to a node on
// the server from which it can be retrieved later.
func Publish(ctx context.Context, s *xmpp.Session, node, id string, item xml.TokenReader) (string, error) {
	return PublishIQ(ctx, s, stanza.IQ{}, node, id, item)
}

// PublishIQ is like Publish except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func PublishIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node, id string, item xml.TokenReader) (string, error) {
	iq.Type = stanza.SetIQ
	start, err := item.Token()
	if err != nil {
		return "", err
	}
	itemAttrs := []xml.Attr{}
	if id != "" {
		itemAttrs = append(itemAttrs, xml.Attr{
			Name:  xml.Name{Local: "id"},
			Value: id,
		})
	}
	resp := publishResponse{}
	err = s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Wrap(
				xmlstream.MultiReader(xmlstream.Token(start), xmlstream.InnerElement(item)),
				xml.StartElement{Name: xml.Name{Local: "item"}, Attr: itemAttrs},
			),
			xml.StartElement{Name: xml.Name{Local: "publish"}, Attr: []xml.Attr{{Name: xml.Name{Local: "node"}, Value: node}}},
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "pubsub"}},
	), stanza.IQ{
		Type: stanza.SetIQ,
	}, &resp)
	if resp.Publish.Item.ID == "" {
		return id, err
	}
	return resp.Publish.Item.ID, err
}
