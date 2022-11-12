// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package blocklist

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Handle returns an option that registers the given handler on the mux for the
// various blocking command payloads.
func Handle(h Handler) mux.Option {
	return func(m *mux.ServeMux) {
		mux.IQ(stanza.GetIQ, xml.Name{Space: NS, Local: "blocklist"}, h)(m)
		mux.IQ(stanza.SetIQ, xml.Name{Space: NS, Local: "block"}, h)(m)
		mux.IQ(stanza.SetIQ, xml.Name{Space: NS, Local: "unblock"}, h)(m)
	}
}

// Handler can be used to respond to incoming blocking command requests.
type Handler struct {
	Block      func(Item)
	Unblock    func(jid.JID)
	UnblockAll func()
	List       func(chan<- jid.JID)
}

// HandleIQ implements mux.IQHandler.
func (h Handler) HandleIQ(iq stanza.IQ, r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	if start.Name.Local == "blocklist" {
		res := iq.Result(xmlstream.Wrap(nil, *start))
		// Copy the start IQ and start payload first.
		_, err := xmlstream.Copy(r, xmlstream.LimitReader(res, 2))
		if err != nil {
			return err
		}
		if h.List != nil {
			c := make(chan jid.JID)
			go func() {
				h.List(c)
				close(c)
			}()
			for j := range c {
				_, err = xmlstream.Copy(r, xmlstream.Wrap(nil, xml.StartElement{
					Name: xml.Name{Space: NS, Local: "item"},
					Attr: []xml.Attr{{
						Name:  xml.Name{Local: "jid"},
						Value: j.String(),
					}},
				}))
				if err != nil {
					return err
				}
			}
		}
		// Copy the end payload and end IQ.
		_, err = xmlstream.Copy(r, xmlstream.LimitReader(res, 2))
		return err
	}

	iter := xmlstream.NewIter(r)
	var found bool
	for iter.Next() {
		found = true
		itemStart, r := iter.Current()
		jstr := itemStart.Attr[0].Value
		j := jid.MustParse(jstr)
		switch start.Name.Local {
		case "block":
			item := Item{}
			d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*itemStart), r))
			if err := d.Decode(&item); err != nil {
				return err
			}
			if h.Block != nil {
				h.Block(item)
			}
		case "unblock":
			if h.Unblock != nil {
				h.Unblock(j)
			}
		}
	}
	err := iter.Err()
	if err != nil {
		return err
	}
	if !found && start.Name.Local == "unblock" && h.UnblockAll != nil {
		h.UnblockAll()
	}
	return nil
}
