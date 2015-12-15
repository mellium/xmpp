package server

import (
	"errors"
	"net"
)

type C2SSession struct {
}

func (h *C2SSession) Handle(c net.Conn, l net.Listener) (err error) {
	defer func() {
		if cerr := c.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	decoder := xml.NewDecoder(c)
	encoder := xml.NewEncoder(c)
	for {
		t, err := decoder.RawToken()
		if err != nil && err != io.EOF {
			return err
		}
		switch t := t.(type) {
		case xml.ProcInst:
			if t.Target != "xml" {
				return errors.New("Received invalid XML procinst")
			}

			err = encoder.EncodeToken(t.Copy())
			if err != nil {
				return
			}
		//case xml.StartElement:
		//	if t.Name == StreamName {
		//		stream, err := StreamFromStartElement(t)

		//		// Send an XML header
		//		_, err = c.Write([]byte(xml.Header))
		//		if err != nil {
		//			return err
		//		}

		//		// TODO(): Validate that we serve the domain in question.

		//		// Create and send a new stream element
		//		s := stream.Copy()
		//		// Swap the to and from attributes from the initiating stream element.
		//		// If a `from' attribute is set, use it. Otherwise, ignore it.
		//		if val, err := stream.From(); err == nil {
		//			s.SetTo(val)
		//		} else {
		//			s.To = &jid.EnforcedJID{}
		//		}
		//		if val, err := stream.To(); err == nil {
		//			s.SetFrom(val)
		//		} else {
		//			return err
		//		}

		//		// Write the new stream
		//		_, err = c.Write(s.Bytes())
		//		if err != nil {
		//			return err
		//		}

		//	} else {
		//		return errors.New("Invalid start element " + t.Name.Local)
		//	}

		default:
			return errors.New("Encountered invalid token while parsing XML")
		}
	}
	return
}
