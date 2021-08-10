// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Handle returns an option that configures a multiplexer to handle service
// discovery requests by iterating over its own handlers and checking if they
// implement the interfaces from the info package.
func Handle() mux.Option {
	return func(m *mux.ServeMux) {
		h := &discoHandler{ServeMux: m}
		mux.IQ(stanza.GetIQ, xml.Name{Space: NSInfo, Local: "query"}, h)(m)
	}
}

type discoHandler struct {
	*mux.ServeMux
}

func (h *discoHandler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return h.ServeMux.HandleXMPP(t, start)
}

func (h *discoHandler) HandleIQ(iq stanza.IQ, r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	seen := make(map[string]struct{})
	pr, pw := xmlstream.Pipe()
	go func() {
		switch start.Name.Space {
		case NSInfo:
			var node string
			for _, attr := range start.Attr {
				if attr.Name.Local == "node" {
					node = attr.Value
					break
				}
			}
			pw.CloseWithError(h.ServeMux.ForFeatures(node, func(f info.Feature) error {
				_, ok := seen[f.Var]
				if ok {
					return nil
				}
				seen[f.Var] = struct{}{}
				_, err := xmlstream.Copy(pw, f.TokenReader())
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
