// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package client

// A Client represents an XMPP client capable of making a single
// client-to-server (C2S) connection on behalf of the configured user.
type Client struct {
	options
}

// New creates a new XMPP client with the given options.
func New(opts ...Option) *Client {
	return &Client{
		options: getOpts(opts...),
	}
}
