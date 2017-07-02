// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"net"
	"testing"
)

var _ error = (*Error)(nil)
var _ error = Error{}
var _ xml.Marshaler = (*Error)(nil)
var _ xml.Marshaler = Error{}
var _ xml.Unmarshaler = (*Error)(nil)

func TestMarshalSeeOtherHost(t *testing.T) {
	for _, test := range []struct {
		ipaddr net.Addr
		xml    string
		err    bool
	}{
		// see-other-host errors should wrap IPv6 addresses in brackets.
		{&net.IPAddr{IP: net.ParseIP("::1")}, `<stream:error><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">[::1]</see-other-host></stream:error>`, false},
		{&net.IPAddr{IP: net.ParseIP("127.0.0.1")}, `<stream:error><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">127.0.0.1</see-other-host></stream:error>`, false},
	} {
		soh := SeeOtherHostError(test.ipaddr)
		xb, err := xml.Marshal(soh)
		switch xbs := string(xb); {
		case test.err && err == nil:
			t.Errorf("Expected marshaling SeeOtherHost error for address `%v` to fail", test.ipaddr)
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case xbs != test.xml:
			t.Logf("Expected `%s` but got `%s`", test.xml, xbs)
			t.Fail()
		}
	}
}

func TestUnmarshal(t *testing.T) {
	for _, test := range []struct {
		xml string
		se  Error
		err bool
	}{
		{
			`<stream:error><restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"></restricted-xml></stream:error>`,
			RestrictedXML, false,
		},
		{
			`<stream:error></a>`,
			RestrictedXML, true,
		},
	} {
		s := Error{}
		err := xml.Unmarshal([]byte(test.xml), &s)
		switch {
		case test.err && err == nil:
			t.Errorf("Expected unmarshaling error for `%v` to fail", test.xml)
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case s.Err != test.se.Err || string(s.InnerXML) != string(test.se.InnerXML):
			t.Errorf("Expected `%#v` but got `%#v`", test.se, s)
		}
	}
}

func TestErrorReturnsErr(t *testing.T) {
	if RestrictedXML.Error() != "restricted-xml" {
		t.Error("Error should return the name of the err")
	}
}
