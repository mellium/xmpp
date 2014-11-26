package xmpp

import (
	"encoding/xml"
	"errors"

	"github.com/SamWhited/koine"
)

// Represents an XMPP stream XML start element.
type Stream struct {
	STo     string   `xml:"to,attr"`
	SFrom   string   `xml:"from,attr"`
	Version string   `xml:"version,attr"`
	Xmlns   string   `xml:"xmlns,attr"`
	Lang    string   `xml:"lang,attr"`
	Id      string   `xml:"id,attr"`
	Name    xml.Name `xml:"http://etherx.jabber.org/streams stream"`
}

// The default XML name of XMPP stream elements.
var NAME xml.Name = xml.Name{Space: "stream", Local: "stream"}

// Fill in stream properties from an XML Start Element.
func (stream *Stream) FromStartElement(start xml.StartElement) error {
	if start.Name != NAME {
		return errors.New(start.Name.Space + ":" + start.Name.Local + " is not a valid start stream tag")
	}

	stream.Name = start.Name

	for _, a := range start.Attr {
		switch a.Name.Local {
		case "to":
			stream.STo = a.Value
		case "from":
			stream.SFrom = a.Value
		case "version":
			stream.Version = a.Value
		case "lang":
			stream.Lang = a.Value
		case "id":
			stream.Id = a.Value
		}
	}

	return nil
}

// Create a copy fo the given stream.
func (stream *Stream) Copy() *Stream {
	s := new(Stream)
	*s = *stream
	return s
}

// Get the `from' attribute of an XMPP stream as a JID.
func (s *Stream) From() (jid.JID, error) {
	return jid.NewJID(s.SFrom)
}

// Get the `to' attribute of an XMPP stream as a JID.
func (s *Stream) To() (jid.JID, error) {
	return jid.NewJID(s.STo)
}

// Convert a stream element to an array of bytes.
func (s *Stream) Bytes() []byte {
	// Ignore errors and just return what we get (if we want an error we can marshal it ourselves)
	out, _ := xml.Marshal(s)
	return out
}

// Convert a stream element to a string.
func (s *Stream) String() string {
	return string(s.Bytes())
}
