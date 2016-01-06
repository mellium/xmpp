// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"errors"

	"bitbucket.org/mellium/xmpp/internal"
	"bitbucket.org/mellium/xmpp/jid"
	"golang.org/x/text/language"
)

// A Stream is a container for the exchange of XML elements between two
// endpoints. It maintains state about stream-level features, and handles
// decoding and routing incoming XMPP stanza's and other elements, as well as
// encoding outgoing XMPP elements. Each XMPP connection has two streams, one
// for input, and one for output.
type Stream struct {
	options
	to      *jid.JID
	from    *jid.JID
	id      string
	version internal.Version
}

// New creates a stream that will be used to initiate a new XMPP connection.
// This should always be used by clients to create a new stream, and by the
// initiating server in server-to-server connections.
func New(to, from *jid.JID, opts ...Option) Stream {
	return Stream{
		to:      to,
		from:    from,
		version: internal.DefaultVersion,
		options: getOpts(opts...),
	}
}

// FromStartElement constructs a new Stream from the given XML StartElement.
func FromStartElement(start xml.StartElement) (Stream, error) {

	stream := Stream{}
	if start.Name.Local != "stream" || start.Name.Space != "stream" {
		return stream, errors.New("Incorrect XML name on stream start element.")
	}

	for _, attr := range start.Attr {
		switch attr.Name {
		case xml.Name{"", "from"}:
			j, err := jid.ParseString(attr.Value)
			if err != nil {
				return stream, err
			}
			stream.from = j
		case xml.Name{"", "to"}:
			j, err := jid.ParseString(attr.Value)
			if err != nil {
				return stream, err
			}
			stream.to = j
		case xml.Name{"", "xmlns"}:
			switch attr.Value {
			case "jabber:server":
				stream.options.s2sStream = true
			case "jabber:client":
				stream.options.s2sStream = false
			default:
				return stream, errors.New("Stream has invalid xmlns.")
			}
		case xml.Name{"xml", "lang"}:
			var err error
			stream.lang, err = language.Parse(attr.Value)
			if err != nil {
				return stream, err
			}
		case xml.Name{"", "id"}:
			stream.id = attr.Value
		case xml.Name{"", "version"}:
			v, err := internal.ParseVersion(attr.Value)
			if err != nil {
				return stream, err
			}
			stream.version = v
		}
	}

	return stream, nil
}

// StartElement creates an XML start element from the given stream which is
// suitable for encoding and transmitting over the wire.
func (s Stream) StartElement() xml.StartElement {
	var xmlns string
	if s.options.s2sStream {
		xmlns = "jabber:server"
	} else {
		xmlns = "jabber:client"
	}
	return xml.StartElement{
		Name: xml.Name{"stream", "stream"},
		Attr: []xml.Attr{
			xml.Attr{
				xml.Name{"", "to"},
				s.to.String(),
			},
			xml.Attr{
				xml.Name{"", "from"},
				s.from.String(),
			},
			xml.Attr{
				xml.Name{"", "version"},
				s.version.String(),
			},
			xml.Attr{
				xml.Name{"xml", "lang"},
				s.options.lang.String(),
			},
			xml.Attr{
				xml.Name{"", "id"},
				s.id,
			},
			xml.Attr{
				xml.Name{"", "xmlns"},
				xmlns,
			},
		},
	}
}
