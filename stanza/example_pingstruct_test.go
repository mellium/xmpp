// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"log"
	"os"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// PingIQ represents a ping from XEP-0199: XMPP Ping.
type PingIQ struct {
	stanza.IQ

	Ping struct{} `xml:"urn:xmpp:ping ping"`
}

func Example_struct() {
	e := xml.NewEncoder(os.Stdout)
	e.Indent("", "\t")

	j := jid.MustParse("feste@example.net/siJo4eeT")
	err := e.Encode(PingIQ{
		IQ: stanza.IQ{
			Type: stanza.GetIQ,
			To:   j,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// <iq id="" to="feste@example.net/siJo4eeT" from="" type="get">
	//	<ping xmlns="urn:xmpp:ping"></ping>
	// </iq>
}
