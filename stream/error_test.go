// Copyright 2015 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stream"
)

var (
	_ error               = (*stream.Error)(nil)
	_ error               = stream.Error{}
	_ xml.Marshaler       = (*stream.Error)(nil)
	_ xml.Marshaler       = stream.Error{}
	_ xml.Unmarshaler     = (*stream.Error)(nil)
	_ xmlstream.Marshaler = (*stream.Error)(nil)
	_ xmlstream.WriterTo  = (*stream.Error)(nil)
)

func TestCompare(t *testing.T) {
	hostGoneApp := stream.HostGone.ApplicationError(xmlstream.Wrap(nil, xml.StartElement{}))
	if !errors.Is(stream.HostGone, hostGoneApp) {
		t.Errorf("did not expect adding application error to affect comparison")
	}
	if errors.Is(stream.HostGone, stream.BadNamespacePrefix) {
		t.Errorf("did not expect two errors with different names to be equivalent")
	}
	if !errors.Is(stream.HostGone, stream.Error{}) {
		t.Errorf("expected empty stream error to compare to any other stream error")
	}
}

var marshalTests = [...]struct {
	se  stream.Error
	xml string
	err bool
}{
	0: {
		// see-other-host errors should wrap IPv6 addresses in brackets.
		se:  stream.SeeOtherHostError(&net.IPAddr{IP: net.ParseIP("::1")}),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">[::1]</see-other-host></error>`,
		err: false,
	},
	1: {
		// see-other-host should not wrap IPv6 addresses in brackets if they are already wrapped.
		se:  stream.SeeOtherHostError(&net.TCPAddr{IP: net.ParseIP("::1"), Port: 5222}),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">[::1]:5222</see-other-host></error>`,
		err: false,
	},
	2: {
		// see-other-host should not mess with IPv4 addresses.
		se:  stream.SeeOtherHostError(&net.IPAddr{IP: net.ParseIP("127.0.0.1")}),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><see-other-host xmlns="urn:ietf:params:xml:ns:xmpp-streams">127.0.0.1</see-other-host></error>`,
		err: false,
	},
	3: {
		se:  stream.UnsupportedEncoding.InnerXML(xmlstream.Token(xml.CharData("test"))),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><unsupported-encoding xmlns="urn:ietf:params:xml:ns:xmpp-streams">test</unsupported-encoding></error>`,
	},
	4: {
		se:  stream.UnsupportedEncoding.ApplicationError(xmlstream.Token(xml.CharData("test"))),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><unsupported-encoding xmlns="urn:ietf:params:xml:ns:xmpp-streams"></unsupported-encoding>test</error>`,
	},
	5: {
		se:  stream.UnsupportedEncoding.ApplicationError(xmlstream.Token(xml.CharData("test"))).InnerXML(xmlstream.Token(xml.CharData("foo"))),
		xml: `<error xmlns="http://etherx.jabber.org/streams"><unsupported-encoding xmlns="urn:ietf:params:xml:ns:xmpp-streams">foo</unsupported-encoding>test</error>`,
	},
}

func TestMarshal(t *testing.T) {
	for i, tc := range marshalTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			xb, err := xml.Marshal(tc.se)
			switch xbs := string(xb); {
			case tc.err && err == nil:
				t.Errorf("expected marshaling to fail")
				return
			case !tc.err && err != nil:
				t.Errorf("did not expect error, got=%v", err)
				return
			case err != nil:
				return
			case xbs != tc.xml:
				t.Errorf("bad output:\nwant=`%s`,\n got=`%s`", tc.xml, xbs)
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
		xml: `<stream:error><restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"></restricted-xml></stream:error>`,
		se:  stream.RestrictedXML,
		err: false,
	},
	1: {
		xml: `<stream:error></a>`,
		se:  stream.RestrictedXML,
		err: true,
	},
}

func TestUnmarshal(t *testing.T) {
	for i, test := range unmarshalTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s := stream.Error{}
			err := xml.Unmarshal([]byte(test.xml), &s)
			switch {
			case test.err && err == nil:
				t.Errorf("expected unmarshaling error for `%v` to fail", test.xml)
				return
			case !test.err && err != nil:
				t.Error(err)
				return
			case err != nil:
				return
			case s.Err != test.se.Err:
				t.Errorf("expected Err `%#v` but got `%#v`", test.se, s)
			}
		})
	}
}

func TestErrorReturnsCondition(t *testing.T) {
	if stream.RestrictedXML.Error() != "restricted-xml" {
		t.Error("error should return the error condition")
	}
}
