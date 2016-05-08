// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"os"

	"bitbucket.org/mellium/xmpp"
	"bitbucket.org/mellium/xmpp/server"
)

func main() {
	st := xmpp.Stanza{ID: "Test"}
	m := xmpp.Message{xmpp.Stanza{ID: "TestM"}}
	fmt.Printf("%+v, %+v\n", m, st)
	os.Exit(0)
	s := server.New()
	s.ListenAndServe()
}
