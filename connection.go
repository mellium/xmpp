package xmpp

import (
	"encoding/xml"
	"errors"
	"io"
	"net"
)

func Handle(c net.Conn, l net.Listener) error {
	var err error
	defer func() {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}()
	decoder := xml.NewDecoder(c)
	encoder := xml.NewEncoder(c)
	_ = encoder
	for {
		t, err := decoder.RawToken()
		if err != nil && err != io.EOF {
			return err
		}
		switch t := t.(type) {
		case xml.ProcInst:
		case xml.StartElement:
			// Jankedy stuff.
			// TODO: Validate that the inst is XML v1.0 and if an encoding is
			// specified that it's UTF-8.
			if t.Name == StreamName {
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

				// Write the new stream
				_, err = c.Write(s.Bytes())
				if err != nil {
					return err
				}

			} else {
				return errors.New("Invalid start element " + t.Name.Local)
			}

		default:
		}
	}
}
