// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package client

import (
	"net"
	"strconv"
	"time"

	"bitbucket.org/mellium/xmpp/jid"
	"bitbucket.org/mellium/xmpp/stream"
)

// A Client represents an XMPP client capable of making a single
// client-to-server (C2S) connection on behalf of the configured JID.
type Client struct {
	options
	jid    *jid.JID
	conn   net.Conn
	input  stream.Stream
	output stream.Stream

	// DNS Cache
	cname   string
	addrs   []*net.SRV
	srvtime time.Time
}

// New creates a new XMPP client with the given options.
func New(j *jid.JID, opts ...Option) *Client {
	return &Client{
		jid:     j.Bare(),
		options: getOpts(opts...),
	}
}

// Connect establishes a connection with the server.
func (c *Client) Connect(password string) error {

	c.options.log.Printf("Establishing C2S connection to %s…\n", c.jid.Domainpart())

	// If the cache has expired, lookup SRV records again.
	if c.srvtime.Add(c.options.srvExpiration).Before(time.Now()) {
		if err := c.LookupSRV(); err != nil {
			return err
		}
	}

	// Try dialing all of the SRV records we know about, breaking as soon as the
	// connection is established.
	var err error
	for _, addr := range c.addrs {
		if conn, e := c.options.dialer.Dial(
			"tcp", net.JoinHostPort(
				addr.Target, strconv.FormatUint(uint64(addr.Port), 10),
			),
		); e != nil {
			err = e
			continue
		} else {
			err = nil
			c.conn = conn
			break
		}
	}
	if err != nil {
		return err
	}

	c.output = stream.New(c.jid.Domain(), c.jid)

	return nil
}

// LookupSRV fetches and caches any xmpp-client SRV records associated with the
// domain name in the clients JID. It is called automatically when a client
// attempts to establish a connection, but can be called manually to force the
// cache to update. If an expiration time is set for the records, LookupSRV
// resets the timeout.
func (c *Client) LookupSRV() error {
	c.options.log.Printf("Refreshing SRV record cache for %s…\n", c.jid.Domainpart())

	if cname, addrs, err := net.LookupSRV(
		"xmpp-client", "tcp", c.jid.Domainpart(),
	); err != nil {
		return err
	} else {
		c.addrs = addrs
		c.cname = cname
	}
	c.srvtime = time.Now()
	return nil
}
