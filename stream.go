package xmpp

import (
	"encoding/xml"
	"errors"

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

var NAME xml.Name = xml.Name{Space: "http://etherx.jabber.org/streams", Local: "stream"}

func NewStream(to jid.JID, from jid.JID, xmlns string, id string) (*Stream, error) {
	stream := new(Stream)
	stream.STo = to.String()
	stream.SFrom = from.String()
	stream.Version = "1.0"
	stream.Id = id
	if xmlns == "jabber:client" || xmlns == "jabber:server" {
		stream.Xmlns = xmlns
	} else {
		return nil, errors.New("Invalid XMLNS")
	}
	stream.Name = NAME

	return stream, nil
}

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

func (stream *Stream) FromString(raw string) error {
	if err := xml.Unmarshal([]byte(raw), &stream); err != nil {
		return err
	}

	return nil
}

func (s *Stream) From() (jid.JID, error) {
	return jid.NewJID(s.SFrom)
}

func (s *Stream) To() (jid.JID, error) {
	return jid.NewJID(s.STo)
}

func (s *Stream) Bytes() []byte {
	// Ignore errors and just return what we get (if we want an error we can marshal it ourselves)
	out, _ := xml.Marshal(s)
	return out
}

func (s *Stream) String() string {
	return string(s.Bytes())
}
