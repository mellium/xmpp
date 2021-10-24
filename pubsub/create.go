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

// CreateNode adds a new node on the pubsub service with the provided
// configuration (or the default configuration if none is provided).
func CreateNode(ctx context.Context, s *xmpp.Session, node string, cfg *form.Data) error {
	return CreateNodeIQ(ctx, s, stanza.IQ{}, node, cfg)
}

// CreateNodeIQ is like Publish except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func CreateNodeIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node string, cfg *form.Data) error {
	iq.Type = stanza.SetIQ
	payload := xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Local: "create"}, Attr: []xml.Attr{{Name: xml.Name{Local: "node"}, Value: node}}},
	)
	if cfg != nil {
		submitted, _ := cfg.Submit()
		payload = xmlstream.MultiReader(payload, xmlstream.Wrap(
			submitted,
			xml.StartElement{Name: xml.Name{Local: "configure"}},
		))
	}

	return s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		payload,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "pubsub"}},
	), iq, nil)
}
