// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco_test

import (
	"crypto/sha1"
	"encoding/xml"
	"hash"
	"strconv"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/muc"
)

var verificationTestCases = [...]struct {
	info disco.Info
	h    hash.Hash
	out  string
}{
	0: {
		info: disco.Info{
			Identity: []info.Identity{{
				Type:     "pc",
				Category: "client",
				Name:     "Exodus 0.9.1",
			}},
			Features: []info.Feature{
				{Var: disco.NSInfo},
				{Var: disco.NSCaps},
				{Var: disco.NSItems},
				{Var: muc.NS},
			},
		},
		h:   sha1.New(),
		out: `QgayPKawpkPSDYmwT/WM94uAlu0=`,
	},
	1: {
		info: disco.Info{
			Identity: []info.Identity{
				{
					Lang:     "en",
					Type:     "pc",
					Category: "client",
					Name:     "Psi 0.11",
				},
				{
					Lang:     "el",
					Type:     "pc",
					Category: "client",
					Name:     "Ψ 0.11",
				},
			},
			Features: []info.Feature{
				{Var: disco.NSInfo},
				{Var: disco.NSCaps},
				{Var: disco.NSItems},
				{Var: muc.NS},
			},
			Form: []form.Data{*form.New(
				form.Hidden("FORM_TYPE", form.Value("urn:xmpp:dataforms:softwareinfo")),
				form.TextMulti("ip_version", form.Value("ipv4"), form.Value("ipv6")),
				form.Text("os", form.Value("Mac")),
				form.Text("os_version", form.Value("10.5.1")),
				form.Text("software", form.Value("Psi")),
				form.Text("software_version", form.Value("0.11")),
			),
			}},
		h:   sha1.New(),
		out: `q07IKJEyjvHSyhy//CH0CxmKi8w=`,
	},
}

func TestVerification(t *testing.T) {
	for i, tc := range verificationTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := tc.info.AppendHash(nil, tc.h)

			if s := string(out); s != tc.out {
				t.Fatalf("wrong hash output: want=%s, got=%s", tc.out, s)
			}
		})
	}

}

// Normally tests call TokenReader by virtue of MarshalXML being implemented in
// terms of WriteXML which is implemented in terms of TokenReader.
// Unfortunately, in this case this isn't true (TokenReader and WriteXML are
// both implemented in terms of an internal function due to error handling
// differences). This type lets us mask out the WriteXML and MarshalXML
// implementations so that the marshal tests always call TokenReader, regardless
// of how MarshalXML is implemented.
type marshalTokenReader struct {
	m xmlstream.Marshaler
}

func (m marshalTokenReader) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := xmlstream.Copy(e, m.m.TokenReader())
	return err
}

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		NoUnmarshal: true,
		Value:       &disco.Caps{},
		Err:         crypto.ErrUnknownAlgo,
	},
	1: {
		NoMarshal: true,
		Value:     &disco.Caps{},
		XML:       `<c xmlns="http://jabber.org/protocol/caps" hash="" node="" ver=""></c>`,
		Err:       crypto.ErrUnknownAlgo,
	},
	2: {
		Value: &disco.Caps{
			Hash: crypto.SHA1,
			Node: "node",
			Ver:  "ver",
		},
		XML: `<c xmlns="http://jabber.org/protocol/caps" hash="sha-1" node="node" ver="ver"></c>`,
	},
	3: {
		NoUnmarshal: true,
		Value: marshalTokenReader{
			m: &disco.Caps{
				Hash: crypto.SHA1,
				Node: "node",
				Ver:  "ver",
			},
		},
		XML: `<c xmlns="http://jabber.org/protocol/caps" hash="sha-1" node="node" ver="ver"></c>`,
	},
	4: {
		NoUnmarshal: true,
		Value: marshalTokenReader{
			m: &disco.Caps{
				Node: "node",
				Ver:  "ver",
			},
		},
		XML: `<c xmlns="http://jabber.org/protocol/caps" node="node" ver="ver"></c>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
