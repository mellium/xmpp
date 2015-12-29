// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/jid"
)

// A stanza represents any top level XMPP stanza (Presence, Message, or IQ)
type Stanza struct {
	Id      string          `xml:"id,attr"`
	Inner   string          `xml:",innerxml"`
	To      jid.EnforcedJID `xml:"to,attr"`
	From    jid.EnforcedJID `xml:"from,attr"`
	Lang    string          `xml:"xml:lang,attr"`
	XmlName xml.Name
}
