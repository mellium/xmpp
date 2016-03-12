// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// TODO(ssw): This package name is just going to cause headaches because it
//            conflicts with the normal websocket package. Possibly rename it.

package websocket

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

func xmlMarshal(v interface{}) (msg []byte, payloadType byte, err error) {
	msg, err = xml.Marshal(v)
	return msg, websocket.TextFrame, err
}

func xmlUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	return xml.Unmarshal(msg, v)
}

// XML is a codec to send/receive XML data in a frame from a WebSocket
// connection.
var XML = websocket.Codec{xmlMarshal, xmlUnmarshal}

// xmppCodec returns a codec to send/receive XMPP a series of WebSocket frames.
// The codecs that are returned have their own stateful XML encoder and decoder.
func xmppCodec() websocket.Codec {
	// TODO: Figure out a good initial buffer size? Let them specify?
	b := bytes.NewBuffer([]byte{})
	e := xml.NewEncoder(b)
	d := xml.NewDecoder(b)
	m := sync.Mutex{}

	d.DefaultSpace = "jabber:client"
	return websocket.Codec{
		Marshal: func(v interface{}) (data []byte, payloadType byte, err error) {
			m.Lock()
			defer b.Reset()
			defer m.Unlock()

			err = e.Encode(v)
			payloadType = websocket.TextFrame
			copy(data, b.Bytes())
			return
		},
		Unmarshal: func(data []byte, payloadType byte, v interface{}) (err error) {
			m.Lock()
			defer b.Reset()
			defer m.Unlock()

			_, err = b.Write(data)
			if err != nil {
				return
			}
			err = d.Decode(v)
			if err != nil {
				return
			}
			return
		},
	}
}

// Handler is a simple interface to a WebSocket browser client that implements
// the XMPP subprotocol for WebSockets as specified in RFC 7395. It checks if
// the origin header is a valid URL by default, and that the value "xmpp" is
// included in config.Protocols (and writes the value to the appropriate header
// in the reply).
type Handler func(*websocket.Conn)

func checkReq(config *websocket.Config, req *http.Request) (err error) {
	config.Origin, err = websocket.Origin(config, req)
	if err == nil && config.Origin == nil {
		return fmt.Errorf("null origin")
	}
	for _, proto := range config.Protocol {
		if proto == "xmpp" {
			return nil
		}
	}
	return fmt.Errorf("not XMPP")
}

// ServeHTTP implements the http.Handler interface for a WebSocket.
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := websocket.Server{Handler: websocket.Handler(h), Handshake: checkReq}
	w.Header().Add("Sec-WebSocket-Protocol", "xmpp")
	s.ServeHTTP(w, req)
}
