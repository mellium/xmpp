// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func ExampleError_MarshalXML() {
	e := xml.NewEncoder(os.Stdout)
	e.Indent("", "\t")

	err := e.Encode(stanza.Error{
		By:        jid.MustParse("me@example.com"),
		Type:      stanza.Cancel,
		Condition: stanza.BadRequest,
		Text: map[string]string{
			"en": "Malformed XML in request",
		},
	})
	if err != nil {
		panic(err)
	}
	// Output:
	// <error type="cancel" by="me@example.com">
	// 	<bad-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></bad-request>
	// 	<text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">Malformed XML in request</text>
	// </error>
}

func ExampleError_UnmarshalXML() {
	d := xml.NewDecoder(strings.NewReader(`
	<error type="cancel" by="me@example.com">
		<bad-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></bad-request>
		<text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">Malformed XML</text>
	</error>`))

	se := stanza.Error{}
	err := d.Decode(&se)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s", se, se.Text[""])
	// Output:
	// bad-request: Malformed XML
}
