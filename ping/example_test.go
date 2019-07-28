// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ping_test

import (
	"encoding/xml"
	"log"
	"os"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ping"
	"mellium.im/xmpp/stanza"
)

func ExampleIQ() {
	j := jid.MustParse("feste@example.net/siJo4eeT")
	e := xml.NewEncoder(os.Stdout)
	e.Indent("", "\t")

	ping := ping.IQ{
		IQ: stanza.IQ{To: j},
	}
	err := e.Encode(ping)
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// <iq id="" to="feste@example.net/siJo4eeT" from="" type="get">
	//	<ping xmlns="urn:xmpp:ping"></ping>
	// </iq>
}
