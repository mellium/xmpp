// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bitbucket.org/mellium/xmpp/jid"
)

// stanza contains fields common to any any top level XMPP stanza (Presence,
// Message, or IQ).
type stanza struct {
	ID    string  `xml:"id,attr"`
	Inner string  `xml:",innerxml"`
	To    jid.JID `xml:"to,attr"`
	From  jid.JID `xml:"from,attr"`
	Lang  string  `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
}
