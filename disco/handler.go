// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Handle returns an option that configures a multiplexer to handle service
// discovery requests by iterating over its own handlers and checking if they
// implement info.FeatureIter, info.IdentityIter, form.Iter, or items.Iter.
func Handle() mux.Option {
	return func(m *mux.ServeMux) {
		h := &discoHandler{ServeMux: m}
		mux.IQ(stanza.GetIQ, xml.Name{Space: NSInfo, Local: "query"}, h)(m)
		mux.IQ(stanza.GetIQ, xml.Name{Space: NSItems, Local: "query"}, h)(m)
	}
}

type discoHandler struct {
	ServeMux *mux.ServeMux
}

func (h *discoHandler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return h.ServeMux.HandleXMPP(t, start)
}

func (h *discoHandler) HandleIQ(iq stanza.IQ, r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	seen := make(map[string]struct{})
	pr, pw := xmlstream.Pipe()
	go func() {
		var node string
		for _, attr := range start.Attr {
			if attr.Name.Local == "node" {
				node = attr.Value
				break
			}
		}
		switch start.Name.Space {
		case NSInfo:
			err := h.ServeMux.ForFeatures(node, func(f info.Feature) error {
				_, ok := seen[f.Var]
				if ok {
					return nil
				}
				seen[f.Var] = struct{}{}
				_, err := xmlstream.Copy(pw, f.TokenReader())
				return err
			})
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			for k := range seen {
				delete(seen, k)
			}
			err = h.ServeMux.ForIdentities(node, func(f info.Identity) error {
				loopKey := f.Category + ":" + f.Type + ":" + f.Name + ":" + f.Lang
				_, ok := seen[loopKey]
				if ok {
					return nil
				}
				seen[loopKey] = struct{}{}
				_, err := xmlstream.Copy(pw, f.TokenReader())
				return err
			})
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			pw.CloseWithError(h.ServeMux.ForForms(node, func(f *form.Data) error {
				result, _ := f.Submit()
				_, err := xmlstream.Copy(pw, result)
				return err
			}))
		case NSItems:
			pw.CloseWithError(h.ServeMux.ForItems(node, func(i items.Item) error {
				_, ok := seen[i.Node]
				if ok {
					return nil
				}
				seen[i.Node] = struct{}{}
				_, err := xmlstream.Copy(pw, i.TokenReader())
				return err
			}))
		}
	}()

	_, err := xmlstream.Copy(r, iq.Result(xmlstream.Wrap(
		pr,
		*start,
	)))
	return err
}
