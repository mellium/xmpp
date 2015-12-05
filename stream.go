package xmpp

import (
	"encoding/xml"
	"errors"

	"bitbucket.org/SamWhited/go-jid"
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
var StreamName xml.Name = xml.Name{Space: "stream", Local: "stream"}

// Fill in stream properties from an XML Start Element.
func StreamFromStartElement(start xml.StartElement) (stream, error) {
	if start.Name != StreamName {
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
func (s *stream) From() (*jid.Jid, error) {
	return jid.FromString(s.SFrom)
}

// Get the `to' attribute of an XMPP stream as a JID.
func (s *stream) To() (*jid.Jid, error) {
	return jid.FromString(s.STo)
}

// Set the `from' attribute of a stream from a jid.Jid.
func (s *stream) SetFrom(j *jid.Jid) {
	s.SFrom = j.String()
}

// Set the `to' attribute of a stream from a jid.Jid.
func (s *stream) SetTo(j *jid.Jid) {
	s.STo = j.String()
}

// Bytes converts String() to a byte array for use with a writer.
func (s *stream) Bytes() []byte {
	return []byte(s.String())
}

// String spits out a valid XML representation without making a call to Marshal (so it's safe to use in MarshalXML)
func (s *stream) String() string {
	var toLine, fromLine string
	if s.STo != "" {
		toLine = "to='" + s.STo + "'"
	} else {
		toLine = ""
	}
	if s.SFrom != "" {
		fromLine = "from='" + s.SFrom + "'"
	} else {
		fromLine = ""
	}

	return "<stream:stream " + toLine + " " + fromLine + " version='" + s.Version + "' xml:lang='" + s.Lang + "' xmlns='" + s.Xmlns + "' xmlns:stream='http://etherx.jabber.org/streams'>"
}
