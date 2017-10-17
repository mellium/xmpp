// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// WrapIQ wraps a payload in an IQ stanza.
// The resulting IQ does not contain an id or from attribute and is thus not
// valid without further processing.
func WrapIQ(to *jid.JID, typ IQType, payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, xml.StartElement{
		Name: xml.Name{Local: "iq"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "to"}, Value: to.String()},
			{Name: xml.Name{Local: "type"}, Value: string(typ)},
		},
	})
}

// WrapMessage wraps a payload in a message stanza.
func WrapMessage(to *jid.JID, typ MessageType, payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, xml.StartElement{
		Name: xml.Name{Local: "message"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "to"}, Value: to.String()},
			{Name: xml.Name{Local: "type"}, Value: string(typ)},
		},
	})
}

// WrapPresence wraps a payload in a presence stanza.
func WrapPresence(to *jid.JID, typ PresenceType, payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, xml.StartElement{
		Name: xml.Name{Local: "presence"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "to"}, Value: to.String()},
			{Name: xml.Name{Local: "type"}, Value: string(typ)},
		},
	})
}
