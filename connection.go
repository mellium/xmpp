package xmpp

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"

	"../config"

	"github.com/SamWhited/logger"
)

func Handle(c net.Conn, l net.Listener) error {
	defer c.Close()
	decoder := xml.NewDecoder(c)
	for {
		t, err := decoder.RawToken()
		if err != nil && err != io.EOF {
			logger.Debug(err.Error())
			return err
		}
		switch t := t.(type) {
		case xml.ProcInst:
			// TODO: Validate that the inst is XML v 1.0 and if an encoding is
			// specified that it's UTF-8.
		case xml.StartElement:
			stream := make(Stream)
			err := stream.FromStartElement(t)
			if err != nil {
				logger.Debug(err.Error())
				return err
			}

			// Check if the stream start is to a host we actually serve
			// TODO: Write a more efficient way to do this for all stanzas.
			err = errors.New("Received stanza for invalid host " + stream.To().String())
			for h := range config.C.Hosts {
				jid, err := jid.NewJID(h.Name)
				if err != nil {
					continue
				}
				if stream.To().Equals(jid) {
					err = nil
					break
				}
			}
			if err != nil {
				logger.Err(err.Debug())
				return err
			}

			// Send back a start element

		default:
			fmt.Println("O:", reflect.TypeOf(t))
		}
	}
}
