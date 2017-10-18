// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ping_test

import (
	"encoding/xml"
	"log"
	"os"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/ping"
)

func Example() {
	j := jid.MustParse("feste@example.net/siJo4eeT")
	e := xml.NewEncoder(os.Stdout)
	e.Indent("", "\t")

	ping := ping.IQ(j)
	if _, err := xmlstream.Copy(e, ping); err != nil {
		log.Fatal(err)
	}
	// Output:
	// <iq type="get" to="feste@example.net/siJo4eeT">
	//	<ping xmlns="urn:xmpp:ping"></ping>
	// </iq>
}
