package xmpp

import (
	"encoding/xml"
	"errors"

	"github.com/SamWhited/koine"
)

// Represents an XMPP stream XML start element.
type stream struct {
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
func StreamFromStartElement(start xml.StartElement) (stream, error) {
	if start.Name != NAME {
		return stream{}, errors.New(start.Name.Space + ":" + start.Name.Local + " is not a valid start stream tag")
	}

	s := new(stream)
	s.Name = start.Name

	for _, a := range start.Attr {
		switch a.Name.Local {
		case "xmlns":
			s.Xmlns = a.Value
		case "to":
			s.STo = a.Value
		case "from":
			s.SFrom = a.Value
		case "version":
			s.Version = a.Value
		case "lang":
			s.Lang = a.Value
		case "id":
			s.Id = a.Value
		}
	}

	return *s, nil
}

// Create a copy fo the given stream.
func (s *stream) Copy() *stream {
	ns := new(stream)
	*ns = *s
	return ns
}

// Get the `from' attribute of an XMPP stream as a JID.
func (s *stream) From() (jid.JID, error) {
	return jid.NewJID(s.SFrom)
}

// Get the `to' attribute of an XMPP stream as a JID.
func (s *stream) To() (jid.JID, error) {
	return jid.NewJID(s.STo)
}

// Set the `from' attribute of a stream from a jid.JID.
func (s *stream) SetFrom(j jid.JID) {
	s.SFrom = j.String()
}

// Set the `to' attribute of a stream from a jid.JID.
func (s *stream) SetTo(j jid.JID) {
	s.STo = j.String()
}

// Convert a stream element to an array of bytes.
func (s *stream) Bytes() []byte {
	// Ignore errors and just return what we get (if we want an error we can marshal it ourselves)
	out, _ := xml.Marshal(s)
	return out
}

// Convert a stream element to a string.
func (s *stream) String() string {
	return string(s.Bytes())
}
