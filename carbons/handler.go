// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package carbons

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

// Handle returns an option that registers a handler for carbon copied messages
// on the multiplexer.
func Handle(h Handler) mux.Option {
	return func(m *mux.ServeMux) {
		recv := xml.Name{Space: NS, Local: "received"}
		mux.Message(stanza.NormalMessage, recv, h)(m)
		mux.Message(stanza.ChatMessage, recv, h)(m)

		sent := xml.Name{Space: NS, Local: "sent"}
		mux.Message(stanza.NormalMessage, sent, h)(m)
		mux.Message(stanza.ChatMessage, sent, h)(m)
	}
}

// Handler can be used to handle incoming carbon copied messages.
type Handler struct {
	F func(m stanza.Message, sent bool, inner xml.TokenReader) error
}

// HandleMessage satisfies mux.MessageHandler.
// it is used by the multiplexer and normally does not need to be called by the
// user.
func (h Handler) HandleMessage(p stanza.Message, r xmlstream.TokenReadEncoder) error {
	// Pop the message start.
	_, err := r.Token()
	if err != nil {
		return err
	}
	iter := xmlstream.NewIter(r)
	for iter.Next() {
		start, child := iter.Current()
		if start.Name.Space == NS && (start.Name.Local == "received" || start.Name.Local == "sent") {
			// Skip the "forwarded" element.
			_, err := child.Token()
			if err != nil {
				return err
			}
			return h.F(p, start.Name.Local == "sent", xmlstream.Inner(child))
		}
	}
	return iter.Err()
}
