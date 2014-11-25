package xmpp

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"reflect"

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
			stream := new(Stream)
			err := stream.FromStartElement(t)
			if err != nil {
				logger.Debug(err.Error())
				return err
			}

			// Send back a start element

		default:
			fmt.Println("O:", reflect.TypeOf(t))
		}
	}
}
