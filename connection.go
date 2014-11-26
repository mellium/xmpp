package xmpp

import (
	"encoding/xml"
	"errors"
	"io"
	"net"
)

func Handle(c net.Conn, l net.Listener) error {
	defer c.Close()
	decoder := xml.NewDecoder(c)
	encoder := xml.NewEncoder(c)
	for {
		t, err := decoder.RawToken()
		if err != nil && err != io.EOF {
			return err
		}
		switch t := t.(type) {
		case xml.ProcInst:
			// TODO: Validate that the inst is XML v 1.0 and if an encoding is
			// specified that it's UTF-8.
		case xml.StartElement:
			if t.Name.Local == "stream" {
				stream, err := StreamFromStartElement(t)

				// Send an XML header
				_, err = c.Write([]byte(xml.Header))
				if err != nil {
					return err
				}

				// TODO: Validate that we serve the domain in question.

				// Create and send a new stream element
				s := stream.Copy()
				// Swap the to and from attributes from the initiating stream element.
				// If a `from' attribute is set, use it. Otherwise, ignore it.
				if val, err := stream.From(); err == nil {
					s.SetTo(val)
				} else {
					s.STo = ""
				}
				if val, err := stream.To(); err == nil {
					s.SetFrom(val)
				} else {
					return err
				}

				// Encode the new stream element.
				// Marshal the new stream element and send it.
				if err := encoder.Encode(s); err != nil {
					return err
				}
			} else {
				return errors.New("Invalid start element " + t.Name.Local)
			}

		default:
		}
	}
}
