# Proposal: Implement XEP-0047: In-Band Bytestreams

**Author(s):** Sam Whited  
**Last updated:** 2020-01-07  
**Status:** thinking
**Discussion:** https://mellium.im/issue/19

## Abstract

An API design is proposed for [XEP-0047: In-Band Bytestreams][XEP-0047].

[XEP-0047]: https://xmpp.org/extensions/xep-0047.html


## Background

In-Band Bytestreams (IBB) is the lowest common denominator for transferring
binary data such as files over XMPP streams.
It is slow and inefficient, but also simple and almost universally supported.
Implementing it will allow us to begin experimenting with multiplexing data over
XMPP streams and provide us with experience designing and implementing file
transfer APIs without the overhead of learning a more complex standard such as
[XEP-0234: Jingle File Transfer][XEP-0234].

[XEP-0234]: https://xmpp.org/extensions/xep-0234.html


## Requirements

 - No external dependencies
 - Ability to create a bidirectional stream (minimal API, no file transfer
   semantics or other extraneous functionality)
 - API must hide underlying XMPP details and provide Go stream semantics


## Proposal

The proposed API creates two new types that would need to remain backwards
compatible after we reach 1.0.

    type Conn struct {
    	// Has unexported fields.
    }
        Conn is an IBB stream. Writes to the stream are buffered up to blocksize and
        calling Close forces any remaining data to be flushed.

    Some methods elided. See [`net.Conn`].

    func (c *Conn) SID() string
        SID returns a unique session ID for the connection.

    func (c *Conn) Size() int
        Size returns the blocksize for the underlying buffer when writing to the IBB
        stream.

    func (c *Conn) Stanza() string
        Stanza returns the carrier stanza type ("message" or "iq") for payloads
        received by the IBB session.

    type Handler struct {
    	// Has unexported fields.
    }
        Handler is an xmpp.Handler that handles multiplexing of bidirectional IBB
        streams.

    func (h *Handler) HandleXMPP(t xmlstream.TokenReadEncoder, start *xml.StartElement) error
        HandleXMPP implements xmpp.Handler.

    func (h *Handler) Open(ctx context.Context, s *xmpp.Session, to jid.JID, blockSize uint16) (*Conn, error)
        Open attempts to create a new IBB stream on the provided session using IQs
        as the carrier stanza.

    func (h *Handler) OpenMessage(ctx context.Context, s *xmpp.Session, to jid.JID, blockSize uint16) (*Conn, error)
        OpenMessage attempts to create a new IBB stream on the provided session
        using messages as the carrier stanza. Most users should call Open instead.


[`net.Conn`]: https://golang.org/pkg/net/#Conn
