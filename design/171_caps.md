# Entity Capabilities

**Author(s):** Sam Whited <sam@samwhited.com>  
**Last updated:** 2021-09-01  
**Discussion:** https://mellium.im/issue/171

## Abstract

An API for maintaining an entity caps hash and sending it on presence
broadcasts.


## Background

Because entity caps is a broadcast protocol and not a request response protocol
like many XMPP extensions we can not simply advertise support for caps by
implementing `info.FeaturesIter` on a handler as we do with most extensions, nor
can we use a handler to respond to request for entity caps since we'd need to
add them to presences and there are no requests.
Instead, caps may need to be implemented as a middleware that wraps the
multiplexer and adds caps to all outgoing presence stanzas and the feature to
all responses to disco#info requests.


## Requirements

- Allow parsing entity caps received as part of a presence
- Calculate arbitrary entity caps hashes given a list of features, identities,
  and forms
- Automatically generate entity caps hashes for a multiplexer based on
  registered handlers
- A basic list of hashes (likely including only the SHA family) must be
  supported by default


## Proposal

    // InsertCaps adds the entity caps string to all presence stanzas read over
    // r.
    func InsertCaps(r xml.TokenReader) xml.TokenReader { … }


Entity capabilities is, in effect, an extension of the service discovery
mechanism. Therefore the following proposal will be implemented in the `disco`
package instead of in its own `caps` package or similar.
Parsing or including caps in a presence will be accomplished with a simple
struct that can be combined with a stanza.Presence.

    // Caps can be included in a presence stanza or in stream features to
    // advertise entity capabilities.
    // Node is a string that uniquely identifies your client (eg.
    // https://example.com/myclient) and ver is the hash of an Info value.
    type Caps struct {
        XMLName xml.Name    `xml:"http://jabber.org/protocol/caps c"`
        Hash    string      `xml:"hash,attr"`
        Node    string      `xml:"node,attr"`
        Ver     string      `xml:"ver,attr"`
    }

    func (Caps) TokenReader() xml.TokenReader {…}
    func (Caps) WriteXML(xmlstream.TokenWriter) (int, error) {…}
    func (Caps) MarshalXML(*xml.Encoder, xml.StartElement) error {…}


Calculating the actual hash will be performed in one of two ways: with a method
on the Info response type and by a function that takes a list of features,
identities, and forms.
Both the method and the function may come in an efficient form for re-using byte
slices as well as a more practical (but slower) string form that allocates the
space it needs:

    // Hash generates the entity capabilities verification string.
    // Its output is suitable for use as a cache key.
    func Hash(hash.Hash, info.FeatureIter, info.IdentityIter, form.Iter) (string, error) { … }

    // AppendHash is like Hash except that it appends the output to the provided
    // byte slice.
    func AppendHash([]byte, hash.Hash, info.FeatureIter, info.IdentityIter, form.Iter) ([]byte, error) { … }

    // Hash generates the entity capabilities verification string.
    // Its output is suitable for use as a cache key.
    func (Info) Hash(h hash.Hash) string {… }

    // AppendHash is like Hash except that it appends the output string to the
    // provided byte slice.
    func (Info) AppendHash(dst []byte, h hash.Hash) []byte { … }

## Open Issues

- Should the user be able to extend/remove hashes from the default list?
- Should `Caps` have some sort of "Verify" method that takes a list of supported
  hashes?
- Should (and "can") we provide a mechanism for advertising entity caps in all
  outgoing presence stanzas?
