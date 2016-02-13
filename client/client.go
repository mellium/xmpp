// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package client

import (
	"bitbucket.org/mellium/xmpp/conn"
	"bitbucket.org/mellium/xmpp/jid"
	"bitbucket.org/mellium/xmpp/stream"
)

// A Client represents an XMPP client capable of making a single
// client-to-server (C2S) connection on behalf of the configured JID.
type Client struct {
	options
	jid    *jid.JID
	conn   *conn.XMPPConn
	input  stream.Stream
	output stream.Stream
}

// New creates a new XMPP client with the given options.
func New(j *jid.JID, opts ...Option) *Client {
	return &Client{
		jid:     j.Bare(),
		options: getOpts(opts...),
	}
}

// Connect establishes a connection with the server.
func (c *Client) Connect(password string) (err error) {

	c.options.log.Printf("Establishing C2S connection to %s…\n", c.jid.Domainpart())
	c.conn, err = conn.Dial(c.jid)

	c.output = stream.New(c.jid.Domain(), c.jid)

	return nil
}
