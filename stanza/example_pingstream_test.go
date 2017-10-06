// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"io"
	"log"
	"os"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// WrapPingIQ returns an xmlstream.TokenReader that outputs a new IQ stanza with
// a ping payload.
func WrapPingIQ(to *jid.JID) xmlstream.TokenReader {
	state := 0
	start := xml.StartElement{Name: xml.Name{Local: "ping", Space: `urn:xmpp:ping`}}
	return stanza.WrapIQ(to, stanza.GetIQ, xmlstream.ReaderFunc(func() (xml.Token, error) {
		switch state {
		case 0:
			state++
			return start, nil
		case 1:
			state++
			return start.End(), io.EOF
		}
		return nil, io.EOF
	}))
}

func Example_stream() {
	j := jid.MustParse("feste@example.net/siJo4eeT")
	e := xml.NewEncoder(os.Stdout)
	e.Indent("", "\t")

	ping := WrapPingIQ(j)
	if err := xmlstream.Copy(e, ping); err != nil {
		log.Fatal(err)
	}
	// Output:
	// <iq to="feste@example.net/siJo4eeT" type="get">
	//	<ping xmlns="urn:xmpp:ping"></ping>
	// </iq>
}
