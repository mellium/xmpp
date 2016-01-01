// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"bytes"
	"encoding/xml"
	"net"
	"testing"
)

var _ error = (*StreamError)(nil)
var _ xml.Marshaler = (*StreamError)(nil)
var _ xml.Unmarshaler = (*StreamError)(nil)

// Both pointers and normal errors should marshal to the same thing.
func TestMarshalPointerAndNormal(t *testing.T) {
	xb, err := xml.Marshal(BadFormat)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	xb2, err := xml.Marshal(&BadFormat)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if len(xb) != len(xb2) {
		t.Log("BadFormat and &BadFormat should marshal identically")
		t.Fail()
	}
}

// see-other-host errors should wrap IPv6 addresses in brackets.
func TestMarshalSeeOtherHostV6(t *testing.T) {
	ipaddr := net.IPAddr{IP: net.ParseIP("::1")}
	soh := SeeOtherHostError(&ipaddr)
	xb, err := xml.Marshal(soh)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if xbs := string(xb); xbs != `<stream:error><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">[::1]</see-other-host></stream:error>` {
		t.Logf("Expected [::1] but got %s", xbs)
		t.Fail()
	}
}

// Stream errors should be marshalable and unmarshalable.
func TestUnmarshalMarshalSteamError(t *testing.T) {
	b := []byte(`<stream:error>
	<restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams">a</restricted-xml>
</stream:error>`)
	mb := bytes.NewBuffer(b)
	d := xml.NewDecoder(mb)
	s := &StreamError{}
	err := d.Decode(s)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if s.Error() != "restricted-xml" {
		t.Logf("Expected restricted-xml but got %+v\n", s)
		t.FailNow()
	}

	xb, err := xml.MarshalIndent(s, "", "\t")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if string(b) != string(xb) {
		t.Logf("Expected %s but got %s", string(b), string(xb))
		t.Fail()
	}
}
