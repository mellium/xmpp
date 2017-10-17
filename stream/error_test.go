// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"encoding/xml"
	"fmt"
	"net"
	"testing"

	"mellium.im/xmpp/stream"
)

var (
	_ error           = (*stream.Error)(nil)
	_ error           = stream.Error{}
	_ xml.Marshaler   = (*stream.Error)(nil)
	_ xml.Marshaler   = stream.Error{}
	_ xml.Unmarshaler = (*stream.Error)(nil)
)

var marshalSeeOtherHostTests = [...]struct {
	ipaddr net.Addr
	xml    string
	err    bool
}{
	// see-other-host errors should wrap IPv6 addresses in brackets.
	0: {&net.IPAddr{IP: net.ParseIP("::1")}, `<error xmlns="http://etherx.jabber.org/streams"><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">[::1]</see-other-host></error>`, false},
	1: {&net.IPAddr{IP: net.ParseIP("127.0.0.1")}, `<error xmlns="http://etherx.jabber.org/streams"><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">127.0.0.1</see-other-host></error>`, false},
}

func TestMarshalSeeOtherHost(t *testing.T) {
	for i, test := range marshalSeeOtherHostTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			soh := stream.SeeOtherHostError(test.ipaddr, nil)
			xb, err := xml.Marshal(soh)
			switch xbs := string(xb); {
			case test.err && err == nil:
				t.Errorf("Expected marshaling SeeOtherHost error for address `%v` to fail", test.ipaddr)
				return
			case !test.err && err != nil:
				t.Error(err)
				return
			case err != nil:
				return
			case xbs != test.xml:
				t.Errorf("Bad output:\nwant=`%s`,\ngot=`%s`", test.xml, xbs)
			}
		})
	}
}

var unmarshalTests = [...]struct {
	xml string
	se  stream.Error
	err bool
}{
	0: {
		`<stream:error><restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"></restricted-xml></stream:error>`,
		stream.RestrictedXML, false,
	},
	1: {
		`<stream:error></a>`,
		stream.RestrictedXML, true,
	},
}

func TestUnmarshal(t *testing.T) {
	for i, test := range unmarshalTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s := stream.Error{}
			err := xml.Unmarshal([]byte(test.xml), &s)
			switch {
			case test.err && err == nil:
				t.Errorf("Expected unmarshaling error for `%v` to fail", test.xml)
				return
			case !test.err && err != nil:
				t.Error(err)
				return
			case err != nil:
				return
			case s.Err != test.se.Err:
				t.Errorf("Expected Err `%#v` but got `%#v`", test.se, s)
				//case string(s.InnerXML) != string(test.se.InnerXML):
				//	t.Errorf("Expected `%#v` but got `%#v`", test.se, s)
			}
		})
	}
}

func TestErrorReturnsErr(t *testing.T) {
	if stream.RestrictedXML.Error() != "restricted-xml" {
		t.Error("Error should return the name of the err")
	}
}
