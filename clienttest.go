// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// +build ignore

package main

import (
	"log"
	"os"

	"bitbucket.org/mellium/xmpp/client"
	"bitbucket.org/mellium/xmpp/jid"
)

func main() {
	j, err := jid.ParseString("sam@samwhited.com")
	if err != nil {
		log.Fatal(err)
	}

	c := client.New(j,
		client.Logger(log.New(os.Stderr, "", log.LstdFlags)))
	c.Connect("test")
	// c.Process()
}
