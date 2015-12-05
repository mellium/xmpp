package xmpp

import (
	"encoding/xml"
)

// A stanza represents any top level XMPP stanza (Presence, Message, or IQ)
type stanza struct {
	Id      string `xml:"id,attr"`
	Inner   string `xml:",innerxml"`
	Sto     string `xml:"to,attr"`
	Sfrom   string `xml:"from,attr"`
	Lang    string `xml:"xml:lang,attr"`
	XmlName xml.Name
}
