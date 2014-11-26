package xmpp

import (
	"encoding/xml"

	"github.com/SamWhited/koine"
)

type Stream struct {
	STo     string   `xml:"to,attr"`
	SFrom   string   `xml:"from,attr"`
	Version string   `xml:"version,attr"`
	Xmlns   string   `xml:"xmlns,attr"`
	Lang    string   `xml:"lang,attr"`
	Id      string   `xml:"id,attr"`
	Name    xml.Name `xml:"http://etherx.jabber.org/streams stream"`
}

var (
	NAME xml.Name = xml.Name{Space: "stream", Local: "stream"}
)

// Create a copy fo the given stream.
func (stream *Stream) Copy() *Stream {
	s := new(Stream)
	*s = *stream
	return s
}

// Get the `from' attribute of a stream as a jid.JID.
func (s *Stream) From() (jid.JID, error) {
	return jid.NewJID(s.SFrom)
}

// Get the `to' attribute of a stream as a jid.JID.
func (s *Stream) To() (jid.JID, error) {
	return jid.NewJID(s.STo)
}

// Set the `from' attribute of a stream from a jid.JID.
func (s *Stream) SetFrom(j jid.JID) {
	s.SFrom = j.String()
}

// Set the `to' attribute of a stream from a jid.JID.
func (s *Stream) SetTo(j jid.JID) {
	s.STo = j.String()
}

// Represent the stream as an array of bytes.
func (s *Stream) Bytes() []byte {
	// Ignore errors and just return what we get (if we want an error we can marshal it ourselves)
	out, _ := xml.Marshal(s)
	return out
}

// Represent the stream as a string.
func (s *Stream) String() string {
	return string(s.Bytes())
}
