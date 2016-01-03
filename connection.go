// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/stream"
)

// ConnManager takes an XML encoder and decoder (probably tied to a TCP
// connection) and launches a goroutine which reads from the output channel and
// encodes (and sends) the result as XML and reads data from the XML decoder and
// writes it to the input channel. The connection manager can be shut down by
// closing the output channel.
func ConnManager(
	encoder *xml.Encoder, decoder *xml.Decoder,
) (in, out chan interface{}) {
	in = make(chan interface{})
	out = make(chan interface{})
	quit := make(chan bool)

	// Encode anything that comes in on the output channel to XML.
	go func() {
		for {
			i, ok := <-out
			if ok {
				encoder.Encode(i)
			} else {
				quit <- true
				break
			}
		}
	}()

	// Decode any input XML and send it to the input channel.
	go func() {
	DecodeLoop:
		for {
			select {
			case <-quit:
				close(quit)
				close(in)
				break DecodeLoop

			default:
				token, err := decoder.RawToken()
				if err != nil {
					break
				}
				switch token.(type) {
				case xml.ProcInst, xml.Comment, xml.Directive, xml.CharData:
					out <- errors.RestrictedXML
				case xml.StartElement:
					// TODO:
					// If stream:stream, handle it.
					// Otherwise, send to input and let the XML router handle it.
				case xml.EndElement:
					// If stream, clean end stream.
					// else, unexpected end element so error and end stream.
				}
				in <- token
			}
		}
	}()

	return in, out
}
