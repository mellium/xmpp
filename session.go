// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"io"
)

// Session represents an XMPP session that can be bound to a connection and
// handles the underlying XML streams. Because sessions are independant of
// connections, the same Session may survive across reconnects and may even be
// switched between different types of connections.
type Session struct {
	in  stream
	out stream

	config *Config
}

// NewSession creates a new XMPP session over the given ReadWriteCloser and
// attempts to authenticate.
func NewSession(config *Config, rwc io.ReadWriteCloser) (*Session, error) {
	return &Session{
		in:  newStream(rwc),
		out: newStream(rwc),

		config: config,
	}, nil
}

type stream struct {
	encoder *xml.Encoder
	decoder *xml.Decoder
}

func newStream(rw io.ReadWriter) stream {
	return stream{
		encoder: xml.NewEncoder(rw),
		decoder: xml.NewDecoder(rw),
	}
}
