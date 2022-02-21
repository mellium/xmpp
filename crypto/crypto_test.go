// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package crypto_test

import (
	stdcrypto "crypto"
	_ "crypto/sha256"
	"encoding/xml"
	"strconv"
	"testing"

	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/internal/xmpptest"
)

var _ stdcrypto.SignerOpts = crypto.SHA256

const badHash = crypto.BLAKE2b_512 + 2

var shouldPanic = [...]func(){
	0: func() { badHash.TokenReader() },
	1: func() { badHash.WriteXML(nil) },
	2: func() { badHash.MarshalXML(nil, xml.StartElement{}) },
	3: func() { badHash.New() },
	4: func() { crypto.HashOutput{Hash: badHash}.TokenReader() },
}

func TestPanics(t *testing.T) {
	for i, tc := range shouldPanic {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected bad function input to panic")
				}
			}()

			tc()
		})
	}
}

func addr(c crypto.Hash) *crypto.Hash {
	return &c
}

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: addr(crypto.SHA256),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha-256"></hash-used>`,
	},
	1: {
		Value: addr(crypto.SHA1),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha-1"></hash-used>`,
	},
	2: {
		Value: addr(crypto.SHA224),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha-224"></hash-used>`,
	},
	3: {
		Value: addr(crypto.SHA384),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha-384"></hash-used>`,
	},
	4: {
		Value: addr(crypto.SHA512),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha-512"></hash-used>`,
	},
	5: {
		Value: addr(crypto.SHA3_256),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha3-256"></hash-used>`,
	},
	6: {
		Value: addr(crypto.SHA3_512),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="sha3-512"></hash-used>`,
	},
	7: {
		Value: addr(crypto.BLAKE2b_256),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="blake2b256"></hash-used>`,
	},
	8: {
		Value: addr(crypto.BLAKE2b_512),
		XML:   `<hash-used xmlns="urn:xmpp:hashes:2" algo="blake2b512"></hash-used>`,
	},
	9: {
		NoMarshal: true,
		Value:     addr(0),
		XML:       `<hash-used xmlns="urn:xmpp:hashes:2"></hash-used>`,
		Err:       crypto.ErrMissingAlgo,
	},
	10: {
		NoMarshal: true,
		Value:     addr(0),
		XML:       `<hash-used xmlns="urn:xmpp:hashes:2" algo="md5"></hash-used>`,
		Err:       crypto.ErrUnknownAlgo,
	},
	11: {
		NoMarshal: true,
		Value:     &crypto.HashOutput{},
		XML:       `<hash xmlns="urn:xmpp:hashes:2" algo="sha-29">dGVzdA==</hash>`,
		Err:       crypto.ErrUnknownAlgo,
	},
	12: {
		Value: &crypto.HashOutput{
			Hash: crypto.SHA384,
			Out:  []byte("test"),
		},
		XML: `<hash xmlns="urn:xmpp:hashes:2" algo="sha-384">dGVzdA==</hash>`,
	},
	13: {
		NoMarshal: true,
		Value: &crypto.HashOutput{
			Hash: crypto.SHA384,
		},
		XML: `<hash xmlns="urn:xmpp:hashes:2" algo="sha-384"></hash>`,
		Err: xml.UnmarshalError("crypto: unexpected XML, expected chardata"),
	},
	14: {
		NoUnmarshal: true,
		Value: &crypto.HashOutput{
			Hash: crypto.SHA384,
		},
		XML: `<hash xmlns="urn:xmpp:hashes:2" algo="sha-384"></hash>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}

func TestUnmarshalAttrBad(t *testing.T) {
	h := addr(0)
	err := h.UnmarshalXMLAttr(xml.Attr{
		Value: "rc4",
	})
	if err == nil {
		t.Fatal("expected unknown algorithm in attr to error")
	}
}

func TestUnknownString(t *testing.T) {
	var h crypto.Hash
	s := h.String()
	const want = "unknown hash value 0"
	if s != want {
		t.Errorf("wrong value for string of uknown hash: want=%q, got=%q", want, s)
	}
}

func TestAvailable(t *testing.T) {
	if !crypto.SHA256.Available() {
		t.Errorf("SHA256 was not reported as available")
	}
	if crypto.BLAKE2b_256.Available() {
		t.Errorf("BLAKE2b was reported as available")
	}
}

func TestHashFunc(t *testing.T) {
	h := crypto.SHA1
	if hf := h.HashFunc(); hf != stdcrypto.SHA1 {
		t.Errorf("wrong hash func: want=%v, got=%v", stdcrypto.SHA1, hf)
	}
}

func TestSize(t *testing.T) {
	if s := crypto.SHA224.Size(); s != stdcrypto.SHA224.Size() {
		t.Errorf("wrong size: want=%d, got=%d", stdcrypto.SHA224.Size(), s)
	}
}

func TestTokenReader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("did not expect panic for hash output with valid hash, got %v", r)
		}
	}()

	crypto.HashOutput{
		Hash: crypto.BLAKE2b_512,
	}.TokenReader()
}

func TestWriteXML(t *testing.T) {
	_, err := crypto.HashOutput{}.WriteXML(nil)
	if err == nil {
		t.Fatalf("expected error with invalid hash output")
	}
}
