// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"bitbucket.org/mellium/xmpp/jid"
)

// The currently supported stream version.
const Version = "1.0"

// A Stream is a container for the exchange of XML elements between two
// endpoints. It maintains state about stream-level features, and handles
// decoding and routing incoming XMPP stanza's and other elements, as well as
// encoding outgoing XMPP elements. Each XMPP connection has two streams, one
// for input, and one for output.
type Stream struct {
	to       jid.JID
	from     jid.JID
	version  string
	xmlns    string
	lang     string
	id       string
	resbound bool
}

// func NewStream(
// 	to, from jid.JID,
// 	lang language.Tag,
// 	f ...StreamFeature) Stream {
// 	return Stream{}
// }

// StreamFromStartElement constructs a new Stream from the given
// xml.StartElement (which must be of the form <stream:stream>).
// func StreamFromStartElement(start xml.StartElement) (*Stream, error) {
//
// 	if start.Name.Local != "stream" || start.Name.Space != "stream" {
// 		return nil, errors.New("Start element must be stream:stream")
// 	}
//
// 	stream := &Stream{}
// 	for _, attr := range start.Attr {
// 		switch attr.Name.Local {
// 		case "from":
// 			j, err := jid.SafeFromString(attr.Value)
// 			if err != nil {
// 				return nil, err
// 			}
// 			stream.from = j
// 		case "to":
// 			j, err := jid.SafeFromString(attr.Value)
// 			if err != nil {
// 				return nil, err
// 			}
// 			stream.to = j
// 		case "xmlns":
// 			stream.xmlns = attr.Value
// 		case "lang":
// 			if attr.Name.Space == "xml" {
// 				stream.lang = attr.Value
// 			}
// 		case "id":
// 			stream.id = attr.Value
// 		}
// 	}
//
// 	return stream, nil
// }
//
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
