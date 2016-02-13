// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package server

import (
	"encoding/xml"

	"bitbucket.org/mellium/xmpp/stream"
)

// StreamManager takes an XML encoder and decoder (probably tied to a TCP
// connection) and launches a goroutine which reads from the output channel and
// encodes (and sends) the result as XML and reads data from the XML decoder and
// writes it to the input channel. The connection manager can be shut down by
// closing the output channel.
func StreamManager(
	encoder *xml.Encoder, decoder *xml.Decoder,
) (in, out chan interface{}) {
	in = make(chan interface{})
	out = make(chan interface{})
	quit := make(chan bool)
	// var currStream stream.Stream

	streamName := xml.Name{"stream", "stream"}

	// Encode anything that comes in on the output channel to XML.
	go func() {
		for {
			i, ok := <-out
			if ok {
				switch t := i.(type) {
				case xml.EndElement:
					if t.Name != streamName {
						encoder.Encode(t)
						break
					}
					encoder.Encode(t)
					close(out)
				case stream.StreamError:
					encoder.Encode(t)
					close(out)
				default:
					encoder.Encode(i)
				}
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
				switch t := token.(type) {
				case xml.ProcInst, xml.Comment, xml.Directive, xml.CharData:
					out <- stream.RestrictedXML
				case xml.StartElement:
					// TODO:
					// If stream:stream, handle it.
					// Otherwise, send to input and let the XML router handle it.
				case xml.EndElement:
					switch t.Name {
					case streamName:
						out <- xml.EndElement{streamName}
					default:
						out <- stream.NotWellFormed
					}
				}
				in <- token
			}
		}
	}()

	return in, out
}
