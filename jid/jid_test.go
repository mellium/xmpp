// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package jid

import (
	"encoding/xml"
	"fmt"
	"net"
	"testing"
)

// Compile time check ot make sure that JID and *JID match several interfaces.
var _ fmt.Stringer = (*JID)(nil)
var _ xml.MarshalerAttr = (*JID)(nil)
var _ xml.UnmarshalerAttr = (*JID)(nil)
var _ net.Addr = (*JID)(nil)

func TestValidJIDs(t *testing.T) {
	for _, jid := range []struct {
		jid, lp, dp, rp string
	}{
		{"example.net", "", "example.net", ""},
		{"example.net/rp", "", "example.net", "rp"},
		{"mercutio@example.net", "mercutio", "example.net", ""},
		{"mercutio@example.net/rp", "mercutio", "example.net", "rp"},
		{"mercutio@example.net/rp@rp", "mercutio", "example.net", "rp@rp"},
		{"mercutio@example.net/rp@rp/rp", "mercutio", "example.net", "rp@rp/rp"},
		{"mercutio@example.net/@", "mercutio", "example.net", "@"},
		{"mercutio@example.net//@", "mercutio", "example.net", "/@"},
		{"mercutio@example.net//@//", "mercutio", "example.net", "/@//"},
	} {
		j, err := Parse(jid.jid)
		switch {
		case err != nil:
			t.Log(err)
			t.Fail()
		case j.Domainpart() != jid.dp:
			t.Logf("Got domainpart %s but expected %s", j.Domainpart(), jid.dp)
			t.Fail()
		case j.Localpart() != jid.lp:
			t.Logf("Got localpart %s but expected %s", j.Localpart(), jid.lp)
			t.Fail()
		case j.Resourcepart() != jid.rp:
			t.Logf("Got resourcepart %s but expected %s", j.Resourcepart(), jid.rp)
			t.Fail()
		}
	}
}

var invalidutf8 = string([]byte{0xff, 0xfe, 0xfd})

func TestInvalidJIDs(t *testing.T) {
	for _, jid := range []string{
		"test@/test",
		invalidutf8 + "@example.com/rp",
		invalidutf8 + "/rp",
		invalidutf8,
		"example.com/" + invalidutf8,
		"lp@/rp",
	} {
		_, err := Parse(jid)
		if err == nil {
			t.Logf("Expected JID %s to fail", jid)
			t.Fail()
		}
	}
}

func TestMarshalEmpty(t *testing.T) {
	attr, err := ((*JID)(nil)).MarshalXMLAttr(xml.Name{})
	switch {
	case err != nil:
		t.Logf("Marshaling an empty JID to an attr should not error but got %v\n", err)
		t.Fail()
	case attr != xml.Attr{}:
		t.Logf("Error marshaling empty JID expected Attr{} but got: %+v\n", err)
		t.Fail()
	}
}

func TestMustParsePanics(t *testing.T) {
	handleErr := func(shouldPanic bool) {
		r := recover()
		switch {
		case shouldPanic && r == nil:
			t.Error("Must parse should panic on invalid JID")
		case !shouldPanic && r != nil:
			t.Error("Must parse should not panic on valid JID")
		}
	}
	for _, t := range []struct {
		jid         string
		shouldPanic bool
	}{
		{"@me", true},
		{"@`me", true},
		{"e@example.net", false},
	} {
		func() {
			defer handleErr(t.shouldPanic)
			MustParse(t.jid)
		}()
	}
}
