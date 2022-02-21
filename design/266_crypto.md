# Cryptographic Hash Functions

**Author(s):** Sam Whited <sam@samwhited.com>  
**Last updated:** 2022-02-22  
**Discussion:** https://mellium.im/issue/266

## Abstract

An API for transmitting commonly used hashing algorithms and their sums.


## Prior Art

- [aioxmpp](https://docs.zombofant.net/aioxmpp/devel/api/public/hashes.html)
- [slixmpp](https://slixmpp.readthedocs.io/en/latest/api/plugins/xep_0300.html)


## Background

Transmitting hash algorithms and their outputs over an XMPP connection has
traditionally be done individually by each individual specification.
However, in 2019 [XEP-0300: Use of Cryptographic Hash Functions in
XMPP][XEP-0300] to provide a common wire format for all hashes to use.
At the same time, a set of recommended functions were split out into [XEP-0414:
Cryptographic Hash Function Recommendations for XMPP][XEP-0414] ("hashrecs").

On the Go side of things, hash functions are expected to implement one of the
interfaces in the [`hash`] package and to register themselves against a
package-scoped registry in the [`crypto`] package.
All of the hashes recommended in hashrecs are implemented in the Go standard
library or the Subrepos and have a corresponding constant in the `crypto`
package.


## Requirements

- Determine hash support and hash a payload using a string received over the
  wire (ie. given "sha-1" generate an actual hash using the SHA-1 algorithm)
- Only require linking supported hash algorithms into the final binary (ie.
  don't import a blake2b implementation when the user may only want to support
  sha3)
- Support listing supported algorithms in disco info responses
- Support marshaling the hash function and the output of the hash function to
  XML


## Proposal

The hash functions themselves will be a list of constants similar to the ones in
the `crypto` package, but with added methods:

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

    func (Hash)  Namespace() (string, error) { … }
    func (Hash)  String() string { … }
    func (Hash)  MarshalXMLAttr(xml.Name) (xml.Attr, error) { … }
    func (*Hash) UnmarshalXMLAttr(xml.Attr) error { … }
    func (*Hash) UnmarshalXML(*xml.Decoder, xml.StartElement) error { … }
    func (Hash)  TokenReader() xml.TokenReader { … }
    func (Hash)  WriteXML(xmlstream.TokenWriter) (int, error) { … }
    func (Hash)  MarshalXML(*xml.Encoder, xml.StartElement) error {
    func (Hash)  Available() bool { … }
    func (Hash)  HashFunc() crypto.Hash { … }
    func (Hash)  New() crypto.Hash { … }

This provides us with a mechanism for transmitting a hash itself, but not the
output of a hash function.
To do this, another type is proposed:

    type HashOutput struct {
      Hash
      Out []byte
    }

    func (HashOutput)  TokenReader() xml.TokenReader { … }
    func (HashOutput)  WriteXML(xmlstream.TokenWriter) (int, error) { … }
    func (HashOutput)  MarshalXML(*xml.Encoder, xml.StartElement) error { … }
    func (*HashOutput) UnmarshalXML(*xml.Decoder, xml.StartElement) error { … }

An alternative design was considered where the New method of Hash would return a
concrete type, `XMLHash` or similar implementing `hash.hash` as well as the
various XML marshaling methods (which would call the `Sum()` method to generate
the final output when marshaled).
However, this design was rejected because unmarshaling into it did not make any
sense and a separate solution would be needed.

We also need a way to advertise support for various hashes during service
discovery.
This will be done explicitly by converting a list of hashes into an iterator
that can be passed to the multiplexer:

    // Features returns an iter that can be registered against a mux to
    // advertise support for the hash list.
    // The iter will return an error for any hashes that are not available in
    // the binary.
    func Features(h ...Hash) info.FeatureIter { … }


[XEP-0300]: https://xmpp.org/extensions/xep-0300.html
[XEP-0414]: https://xmpp.org/extensions/xep-0414.html
[`hash`]: https://pkg.go.dev/hash
[`crypto`]: https://pkg.go.dev/crypto
[`hash.Hash`]: https://pkg.go.dev/hash#Hash
[`xtime`]: https://pkg.go.dev/mellium.im/xmpp/xtime
