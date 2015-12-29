// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"

	"bitbucket.org/mellium/xmpp"
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

			// Write an XML header
			_, err = c.Write([]byte(xml.Header))
			if err != nil {
				return err
			}
		case xml.StartElement:
			if t.Name.Local == "stream" && t.Name.Space == "stream" {
				stream, err := xmpp.StreamFromStartElement(t)
				if err != nil {
					return err
				}

				return stream.Handle(encoder, decoder)
			} else {
				return errors.New(fmt.Sprintf("Invalid start element %s", t.Name))
			}
		default:
			return errors.New("Encountered invalid token while parsing XML")
		}
	}
	return
}
