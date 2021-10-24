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

// Delete removes an item from the pubsub node.
func Delete(ctx context.Context, s *xmpp.Session, node, id string, notify bool) error {
	return DeleteIQ(ctx, s, stanza.IQ{}, node, id, notify)
}

// DeleteIQ is like Publish except that it allows modifying the IQ.
// Changes to the IQ type will have no effect.
func DeleteIQ(ctx context.Context, s *xmpp.Session, iq stanza.IQ, node, id string, notify bool) error {
	iq.Type = stanza.SetIQ
	retractAttrs := []xml.Attr{{Name: xml.Name{Local: "node"}, Value: node}}
	if notify {
		retractAttrs = append(retractAttrs, xml.Attr{
			Name:  xml.Name{Local: "notify"},
			Value: "true",
		})
	}
	return s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			xmlstream.Wrap(
				nil,
				xml.StartElement{Name: xml.Name{Local: "item"}, Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: id}}},
			),
			xml.StartElement{Name: xml.Name{Local: "retract"}, Attr: retractAttrs},
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "pubsub"}},
	), stanza.IQ{
		Type: stanza.SetIQ,
	}, nil)
}
