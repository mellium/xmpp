// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature

// Package crypto contains common cryptographic elements.
package crypto // import "mellium.im/xmpp/crypto"

import (
	"crypto"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"strconv"

	"mellium.im/xmlstream"
)

// NS is the namespace used by this package.
const NS = "urn:xmpp:hashes:2"

// A list of errors returned by functions in this package.
// Error checking against these errors should always use errors.Is and not a
// direct comparison.
var (
	ErrMissingAlgo  = errors.New("crypto: no algo attr found")
	ErrUnknownAlgo  = errors.New("crypto: unknown hash value")
	ErrUnlinkedAlgo = errors.New("crypto: attempted to use a hash function without an implementation linked in")
)

// Hash identifies a cryptographic hash function that is implemented in another
// package.
// It is like crypto/hash from the standard library, except only hash functions
// commonly supported in XMPP are given names and values have methods that are
// useful for communicating information about supported hashes over the wire.
type Hash crypto.Hash

// A list of commonly supported hashes and the imports required to enable them.
const (
	SHA1        = Hash(crypto.SHA1)        // import crypto/sha1
	SHA224      = Hash(crypto.SHA224)      // import crypto/sha256
	SHA256      = Hash(crypto.SHA256)      // import crypto/sha256
	SHA384      = Hash(crypto.SHA384)      // import crypto/sha512
	SHA512      = Hash(crypto.SHA512)      // import crypto/sha512
	SHA3_256    = Hash(crypto.SHA3_256)    // import golang.org/x/crypto/sha3
	SHA3_512    = Hash(crypto.SHA3_512)    // import golang.org/x/crypto/sha3
	BLAKE2b_256 = Hash(crypto.BLAKE2b_256) // import golang.org/x/crypto/blake2b
	BLAKE2b_512 = Hash(crypto.BLAKE2b_512) // import golang.org/x/crypto/blake2b
)

// Namespace returns a unique URN for the hash algorithm.
// If the hash algorithm is unknown, Namespace returns an error.
func (h Hash) Namespace() (string, error) {
	switch h {
	case SHA1:
		return "urn:xmpp:hash-function-text-names:sha-1", nil
	case SHA224:
		return "urn:xmpp:hash-function-text-names:sha-224", nil
	case SHA256:
		return "urn:xmpp:hash-function-text-names:sha-256", nil
	case SHA384:
		return "urn:xmpp:hash-function-text-names:sha-384", nil
	case SHA512:
		return "urn:xmpp:hash-function-text-names:sha-512", nil
	case SHA3_256:
		return "urn:xmpp:hash-function-text-names:sha3-256", nil
	case SHA3_512:
		return "urn:xmpp:hash-function-text-names:sha3-512", nil
	case BLAKE2b_256:
		return "urn:xmpp:hash-function-text-names:id-blake2b256", nil
	case BLAKE2b_512:
		return "urn:xmpp:hash-function-text-names:id-blake2b512", nil
	default:
		return "", fmt.Errorf("%w %d", ErrUnknownAlgo, h)
	}
}

// MarshalXMLAttr implements xml.MarshalerAttr.
func (h Hash) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	switch h {
	case SHA1, SHA224, SHA256, SHA384, SHA512, SHA3_256, SHA3_512, BLAKE2b_256, BLAKE2b_512:
	default:
		return xml.Attr{}, fmt.Errorf("%w %d", ErrUnknownAlgo, h)
	}
	return xml.Attr{
		Name:  name,
		Value: h.String(),
	}, nil
}

// Parse creates a hash from the hash name as a string.
func Parse(name string) (Hash, error) {
	switch name {
	case "sha-1":
		return SHA1, nil
	case "sha-224":
		return SHA224, nil
	case "sha-256":
		return SHA256, nil
	case "sha-384":
		return SHA384, nil
	case "sha-512":
		return SHA512, nil
	case "sha3-256":
		return SHA3_256, nil
	case "sha3-512":
		return SHA3_512, nil
	case "blake2b256":
		return BLAKE2b_256, nil
	case "blake2b512":
		return BLAKE2b_512, nil
	}
	return 0, fmt.Errorf("%w %s", ErrUnknownAlgo, name)
}

// UnmarshalXMLAttr implements xml.UnmarshalerAttr.
func (h *Hash) UnmarshalXMLAttr(attr xml.Attr) error {
	newHash, err := Parse(attr.Value)
	if err != nil {
		return err
	}
	*h = newHash
	return nil
}

// UnmarshalXML implements xml.Unmarshaler.
func (h *Hash) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	err := unmarshalXML(h, d, start)
	if err != nil {
		return err
	}
	return d.Skip()
}

func unmarshalXML(h *Hash, d *xml.Decoder, start xml.StartElement) error {
	var found bool
	for _, attr := range start.Attr {
		if attr.Name.Local == "algo" {
			err := h.UnmarshalXMLAttr(attr)
			if err != nil {
				return err
			}
			found = true
			break
		}
	}
	if !found {
		return ErrMissingAlgo
	}
	return nil
}

// TokenReader implements xmlstream.Marshaler.
// TokenReader panics if the hash is invalid.
func (h Hash) TokenReader() xml.TokenReader {
	attr, err := h.MarshalXMLAttr(xml.Name{Local: "algo"})
	if err != nil {
		panic(err)
	}
	return xmlstream.Wrap(
		nil,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "hash-used"},
			Attr: []xml.Attr{attr},
		},
	)
}

// WriteXML implements xmlstream.WriterTo.
// WriteXML panics if the hash is invalid.
func (h Hash) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, h.TokenReader())
}

// MarshalXML implements xml.Marshaler.
// MarshalXML panics if the hash is invalid.
func (h Hash) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := h.WriteXML(e)
	return err
}

// Available reports whether the given hash function is linked into the binary.
func (h Hash) Available() bool {
	return crypto.Hash(h).Available()
}

// HashFunc returns the hash as a crypto.Hash and implements crypto.SignerOpts.
func (h Hash) HashFunc() crypto.Hash {
	return crypto.Hash(h)
}

// New returns a new hash.Hash calculating the given hash function.
// New panics if the hash is invalid.
func (h Hash) New() hash.Hash {
	return crypto.Hash(h).New()
}

// Size returns the length, in bytes, of a digest resulting from the given hash
// function.
// It doesn't require that the hash function in question be linked into the
// program.
func (h Hash) Size() int {
	return crypto.Hash(h).Size()
}

// String implements fmt.Stringer by returning the name of the hash as it would
// appear in wire format.
// This is different from the value returned by the String method of
// crypto.Hash.
func (h Hash) String() string {
	switch h {
	case SHA1:
		return "sha-1"
	case SHA224:
		return "sha-224"
	case SHA256:
		return "sha-256"
	case SHA384:
		return "sha-384"
	case SHA512:
		return "sha-512"
	case SHA3_256:
		return "sha3-256"
	case SHA3_512:
		return "sha3-512"
	case BLAKE2b_256:
		return "blake2b256"
	case BLAKE2b_512:
		return "blake2b512"
	default:
		return "unknown hash value " + strconv.Itoa(int(h))
	}
}

// HashOutput is used to marshal or unmarshal the results of a hash calculation.
type HashOutput struct {
	Hash Hash
	Out  []byte
}

// TokenReader implements xmlstream.Marshaler.
// TokenReader panics if the original hash is invalid.
func (h HashOutput) TokenReader() xml.TokenReader {
	tr, err := tokenReader(h)
	if err != nil {
		panic(err)
	}
	return tr
}

func tokenReader(h HashOutput) (xml.TokenReader, error) {
	attr, err := h.Hash.MarshalXMLAttr(xml.Name{Local: "algo"})
	if err != nil {
		return nil, err
	}
	l := base64.StdEncoding.EncodedLen(len(h.Out))
	out := make([]byte, l)
	base64.StdEncoding.Encode(out, h.Out)
	return xmlstream.Wrap(
		xmlstream.Token(xml.CharData(out)),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "hash"},
			Attr: []xml.Attr{attr},
		},
	), nil
}

// WriteXML implements xmlstream.WriterTo.
func (h HashOutput) WriteXML(w xmlstream.TokenWriter) (int, error) {
	tr, err := tokenReader(h)
	if err != nil {
		return 0, err
	}
	return xmlstream.Copy(w, tr)
}

// MarshalXML implements xml.Marshaler.
func (h HashOutput) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := h.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (h *HashOutput) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	err := unmarshalXML(&h.Hash, d, start)
	if err != nil {
		return err
	}
	tok, err := d.Token()
	if err != nil {
		return err
	}
	charData, ok := tok.(xml.CharData)
	if !ok {
		return xml.UnmarshalError("crypto: unexpected XML, expected chardata")
	}
	l := base64.StdEncoding.DecodedLen(len(charData))
	if len(h.Out) < l {
		h.Out = append(h.Out, make([]byte, l-len(h.Out))...)
	}
	n, err := base64.StdEncoding.Decode(h.Out, charData)
	if err != nil {
		return err
	}
	h.Out = h.Out[:n]
	return d.Skip()
}
