// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp_test

import (
	"context"
	"log"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

// This example uses the raw connection and an XML encoder to send a message.
// Most users will want to use a higher level API.

var (
	laddr = jid.MustParse("feste@shakespeare.lit")
	raddr = jid.MustParse("olivia@example.net")
)

const password = "supersecretpassword"

func Example_rawSendMessage() {
	config := xmpp.NewClientConfig(
		laddr,
		xmpp.StartTLS(true),
		xmpp.SASL(sasl.ScramSha256Plus, sasl.ScramSha256, sasl.Plain),
		xmpp.BindResource(),
	)
	config.Password = password

	log.Printf("Dialing upstream XMPP server as %s…\n", laddr)

	c, err := xmpp.DialConfig(context.Background(), "tcp", config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Connected with JID `%s`\n", c.LocalAddr())

	err = c.Encoder().Encode(struct {
		xmpp.Message
		Body string `xml:"body"`
	}{
		Message: xmpp.Message{
			ID:   "1234",
			To:   raddr,
			From: c.LocalAddr().(*jid.JID),
		},
		Body: "Mercury endue thee with leasing, for thou speakest well of fools!",
	})
	if err != nil {
		log.Fatal(err)
	}

	err = c.Encoder().Flush()
	if err != nil {
		log.Fatal(err)
	}
}
