package xmpp

import (
	"encoding/xml"

	"bitbucket.org/SamWhited/go-jid"
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
