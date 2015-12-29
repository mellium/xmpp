// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package server

import (
	"crypto/tls"
)

// Option's can be used to configure the server.
type Option func(*options)
type options struct {
	clientAddr string // TCP address to listen on, ":xmpp-client" if empty.
	// TODO: Figure out how we want to handle vhosts for the server API
	vhosts    []string
	tlsConfig *tls.Config
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}
	return
}

// The ClientAddr option sets the interface and port that the server will listen
// on for inbounc connections from XMPP clients.
func ClientAddr(addr string) Option {
	return func(o *options) {
		o.clientAddr = addr
	}
}

// The TLS option fully configures the servers TLS including the certificate
// chains used, cipher suites, etc. based on the given tls.Config.
func TLS(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}

var (
	PreferClientCipherSuites = preferClientCipherSuites // Prefer cipher suite order indicated by the client (not recommended).
)

var preferClientCipherSuites = func(o *options) {
	if o.tlsConfig == nil {
		o.tlsConfig = &tls.Config{}
	}
	o.tlsConfig.PreferServerCipherSuites = true
}
