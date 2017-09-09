// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"log"
	"os"

	"mellium.im/xmpp/stanza"
)

// PingIQ represents a ping or ping response from XEP-0199: XMPP Ping.
type PingIQ struct {
	stanza.IQ

	Ping struct{} `xml:"urn:xmpp:ping ping"`
}

func Example_pingStruct() {
	e := xml.NewEncoder(os.Stdout)
	err := e.Encode(PingIQ{
		IQ: stanza.IQ{Type: stanza.GetIQ},
	})
	if err != nil {
		log.Fatal(err)
	}
	// Output: <iq id="" type="get"><ping xmlns="urn:xmpp:ping"></ping></iq>
}
