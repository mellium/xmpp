// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"bitbucket.org/mellium/xmpp/jid"
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
	version Version
	id      string
	authed  bool
	bound   bool
}

// New creats an XMPP stream that will be used to initiate a new XMPP
// connection. This should always be used by clients to create a new stream, and
// by the initiating server in a server-to-server connection.
func New(to, from jid.JID, opts ...Option) Stream {
	return Stream{
		to:      *to,
		from:    *from,
		version: DefaultVersion,
		options: getOpts(opts...),
	}
}

// ReadResponse constructs a new Steam that acts as a response stream to the
// provided initiating stream. Respond should always be used by clients and
// servers to to initiate new streams after receiving a client-to-server or
// server-to-server connection.
func ReadResponse(responds, replaces *Stream, opts ...Option) (Stream, error) {
	s := Stream{
		options: getOpts(opts...),
	}

	switch {
	case replaces != nil:
		s.version = replaces.Version
	// RFC 6120 §4.7.5  version.
	//    2.  The receiving entity MUST set the value of the 'version'
	//        attribute in the response stream header to either the value
	//        supplied by the initiating entity or the highest version number
	//        supported by the receiving entity, whichever is lower.
	case replaces == nil && responds.version.Less(DefaultVersion):
		s.version = responds.version
	default:
		s.version = DefaultVersion
	}

	if responds.from != nil {
		s.to = responds.from
	}

	if responds.to != nil {
		// TODO: Verify that we serve this domain, possibly set this to a canonical
		// domain if there is one.
		s.from = responds.to
	}

	return s, nil
}

// fromStartElement constructs a new Stream from the given xml.StartElement.
func fromStartElement(start xml.StartElement) (Stream, error) {

	if start.Name.Local != "stream" || start.Name.Space != "stream" {
		return nil, errors.New("Incorrect XML name on stream start element.")
	}

	stream := Stream{}
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "from":
			j, err := jid.SafeFromString(attr.Value)
			if err != nil {
				return nil, err
			}
			stream.from = j
		case "to":
			j, err := jid.SafeFromString(attr.Value)
			if err != nil {
				return nil, err
			}
			stream.to = j
		case "xmlns":
			stream.xmlns = attr.Value
		case "lang":
			if attr.Name.Space == "xml" {
				stream.lang = attr.Value
			}
		case "id":
			stream.id = attr.Value
		}
	}

	return stream, nil
}

// // StartElement creates an XML start element from the given stream which is
// // suitable for starting an XMPP stream.
// func (s *Stream) StartElement() xml.StartElement {
// 	return xml.StartElement{
// 		Name: xml.Name{"stream", "stream"},
// 		Attr: []xml.Attr{
// 			xml.Attr{
// 				xml.Name{"", "to"},
// 				s.to.String(),
// 			},
// 			xml.Attr{
// 				xml.Name{"", "from"},
// 				s.from.String(),
// 			},
// 			xml.Attr{
// 				xml.Name{"", "version"},
// 				s.version,
// 			},
// 			xml.Attr{
// 				xml.Name{"xml", "lang"},
// 				s.lang,
// 			},
// 			xml.Attr{
// 				xml.Name{"", "id"},
// 				s.id,
// 			},
// 			xml.Attr{
// 				xml.Name{"", "xmlns"},
// 				s.xmlns,
// 			},
// 		},
// 	}
// }

// func (s *Stream) Handle(encoder *xml.Encoder, decoder *xml.Decoder) error {
// 	return errors.New("Test me")
// }
