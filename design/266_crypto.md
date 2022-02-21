# Cryptographic Hash Functions

**Author(s):** Sam Whited <sam@samwhited.com>  
**Last updated:** 2022-02-20  
**Discussion:** https://mellium.im/issue/266

## Abstract

An API for transmitting commonly used hashing algorithms and their sums.


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

    func (Hash) Namespace() string { … }
    func (Hash) String() string { … }
    func (Hash) MarshalXMLAttr(xml.Name) (xml.Attr, error) { … }
    func (Hash) TokenReader() xml.TokenReader { … }
    func (Hash) WriteXML(xmlstream.TokenWriter) (int, error) { … }
    func (Hash) MarshalXML(*xml.Encoder, xml.StartElement) error {
    func (h Hash) New() XMLHash { … }

Like the [`New` method] on `crypto.Hash` we provide a new method for actually
getting a value that can be used to generate the final hash output.
However, ours hash type also implements the various XML marshaling methods:

    type XMLHash struct {
      hash.Hash
    }

    func (XMLHash) TokenReader() xml.TokenReader { … }
    func (XMLHash) WriteXML(xmlstream.TokenWriter) (int, error) { … }
    func (XMLHash) MarshalXML(*xml.Encoder, xml.StartElement) error { … }


## Open Issues

- Should this package be renamed to `xcrypto` similar to [`xtime`] to avoid the
  name conflict with `crypto` in the standard library?
- Should the service discovery feature handle all linked in hashes, or an
  explicitly supplied list of hashes?
- What does the service discovery feature look like?

[XEP-0300]: https://xmpp.org/extensions/xep-0300.html
[XEP-0414]: https://xmpp.org/extensions/xep-0414.html
[`hash`]: https://pkg.go.dev/hash
[`crypto`]: https://pkg.go.dev/crypto
[`New` method]: https://pkg.go.dev/crypto#Hash.New
[`hash.Hash`]: https://pkg.go.dev/hash#Hash
[`xtime`]: https://pkg.go.dev/mellium.im/xmpp/xtime
