// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

// Response is the response to a command.
// It may contain other commands that can be execute in sequence.
type Response struct {
	stanza.IQ

	Node   string `xml:"node,attr"`
	SID    string `xml:"sessionid,attr"`
	Status string `xml:"status,attr"`
}

// Cancel ends the multi-stage command.
func (r Response) Cancel() Command {
	return Command{
		JID:    r.IQ.From,
		SID:    r.SID,
		Node:   r.Node,
		Action: "cancel",
	}
}

// Complete ends the multi-stage command and optionally submits data.
func (r Response) Complete() Command {
	return Command{
		JID:    r.IQ.From,
		SID:    r.SID,
		Node:   r.Node,
		Action: "complete",
	}
}

// Next requests the next step in a multi-stage command.
func (r Response) Next() Command {
	return Command{
		JID:    r.IQ.From,
		SID:    r.SID,
		Node:   r.Node,
		Action: "next",
	}
}

// Prev requests the previous step in a multi-stage command.
func (r Response) Prev() Command {
	return Command{
		JID:    r.IQ.From,
		SID:    r.SID,
		Node:   r.Node,
		Action: "prev",
	}
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (r Response) TokenReader() xml.TokenReader {
	return r.IQ.Wrap(xmlstream.Wrap(
		nil,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "command"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "node"}, Value: r.Node},
				{Name: xml.Name{Local: "sessionid"}, Value: r.SID},
				{Name: xml.Name{Local: "status"}, Value: r.Status},
			},
		},
	))
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (r Response) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, r.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (r Response) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := r.WriteXML(e)
	return err
}
