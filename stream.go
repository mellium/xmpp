package xmpp

import (
	"errors"

	"bitbucket.org/mellium/xmpp/jid"
)

type Stream struct {
	to, from *jid.EnforcedJID
	version  string
	xmlns    string
	lang     string
	id       string
}

func (stream *Stream) Handle(
	encoder xml.Encoder, decoder xml.Decoder,
) error {
}

// StreamFromStartElement constructs a new Stream from the given
// xml.StartElement (which must be of the form <stream:stream>).
func StreamFromStartElement(
	encoder xml.Encoder, decoder xml.Decoder, start *xml.StartElement,
) (*Stream, error) {

	if start.Name.Local != "stream" || start.Name.Space != "stream" {
		return nil, errors.New("Start element must be stream:stream")
	}

	stream := &Stream{}
	for attr := range start.Attr {
		switch attr.Name.Local {
		case "from":
			j, err = jid.EnforcedFromString(attr.Value)
			if err != nil {
				return nil, err
			}
			stream.from = j
		case "to":
			j, err = jid.EnforcedFromString(attr.Value)
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
}

func (*Stream) StartElement() xml.StartElement {
	return xml.StartElement{
		Name: xml.Name{"stream", "stream"},
		Attr: []xml.Attr{},
	}
}
