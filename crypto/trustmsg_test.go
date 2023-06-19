// Copyright 2023 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package crypto_test

import (
	"encoding/base64"
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
)

var (
	_ xml.Marshaler       = crypto.Key{}
	_ xml.Unmarshaler     = (*crypto.Key)(nil)
	_ xmlstream.Marshaler = crypto.Key{}
	_ xmlstream.WriterTo  = crypto.Key{}
	_ xml.Marshaler       = crypto.OwnedKeys{}
	_ xml.Unmarshaler     = (*crypto.OwnedKeys)(nil)
	_ xmlstream.Marshaler = crypto.OwnedKeys{}
	_ xmlstream.WriterTo  = crypto.OwnedKeys{}
	//_ xml.Marshaler       = crypto.TrustMessage{}
	//_ xml.Unmarshaler     = (*crypto.TrustMessage)(nil)
	//_ xmlstream.Marshaler = crypto.TrustMessage{}
	//_ xmlstream.WriterTo  = crypto.TrustMessage{}
)

var tmMarshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &crypto.Key{},
		XML:   `<distrust></distrust>`,
	},
	1: {
		Value: &crypto.Key{Trusted: true},
		XML:   `<trust></trust>`,
	},
	2: {
		Value: &crypto.Key{Trusted: true, KeyID: []byte("123")},
		XML:   `<trust>MTIz</trust>`,
	},
	3: {
		Value: &crypto.Key{KeyID: []byte("abcd")},
		XML:   `<distrust>YWJjZA==</distrust>`,
	},
	4: {
		XML:       `<distrust>YWJjZA==<foo/></distrust>`,
		Value:     &crypto.Key{},
		NoMarshal: true,
		Err:       base64.CorruptInputError(8),
	},
	5: {
		XML:   `<key-owner jid=""></key-owner>`,
		Value: &crypto.OwnedKeys{},
	},
	6: {
		XML:   `<key-owner jid="bob@example.com"></key-owner>`,
		Value: &crypto.OwnedKeys{Owner: jid.MustParse("bob@example.com")},
	},
	7: {
		XML: `<key-owner jid=""><trust>YWJj</trust><distrust>MTIz</distrust></key-owner>`,
		Value: &crypto.OwnedKeys{
			Keys: []crypto.Key{
				{Trusted: true, KeyID: []byte("abc")},
				{Trusted: false, KeyID: []byte("123")},
			},
		},
	},
	8: {
		// Test that we don't re-order keys.
		XML: `<key-owner jid=""><distrust>MTIz</distrust><trust>YWJj</trust></key-owner>`,
		Value: &crypto.OwnedKeys{
			Keys: []crypto.Key{
				{Trusted: false, KeyID: []byte("123")},
				{Trusted: true, KeyID: []byte("abc")},
			},
		},
	},
	9: {
		// Ensure that an error is returned if any non-key winds up in the list.
		XML:       `<key-owner jid=""><distrust>MTIz</distrust><trust>YWJj</trust><foo/></key-owner>`,
		NoMarshal: true,
		Value:     &crypto.OwnedKeys{},
		Err:       crypto.ErrTrustElement,
	},
	10: {
		Value: &crypto.TrustMessage{},
		XML:   `<trust-message xmlns="urn:xmpp:tm:1" usage="" encryption=""></trust-message>`,
	},
	11: {
		Value: &crypto.TrustMessage{
			Usage:      "foo",
			Encryption: "bar",
		},
		XML: `<trust-message xmlns="urn:xmpp:tm:1" usage="foo" encryption="bar"></trust-message>`,
	},
	12: {
		Value: &crypto.TrustMessage{
			Keys: []crypto.OwnedKeys{
				{
					Owner: jid.MustParse("bob@example.net"),
					Keys: []crypto.Key{
						{KeyID: []byte("123")},
					},
				},
			},
		},
		XML: `<trust-message xmlns="urn:xmpp:tm:1" usage="" encryption=""><key-owner jid="bob@example.net"><distrust>MTIz</distrust></key-owner></trust-message>`,
	},
}

func TestTMEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, tmMarshalTestCases)
}
