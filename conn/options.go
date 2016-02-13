// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package conn

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"time"

	"bitbucket.org/mellium/xmpp/jid"
)

// Option's can be used to configure the connection.
type Option func(*options)
type options struct {
	log           *log.Logger
	tlsConfig     *tls.Config
	srvExpiration time.Duration
	dialer        net.Dialer
	network       string
	raddr         *jid.JID
}

func getOpts(laddr *jid.JID, o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}

	// Log to /dev/null by default.
	if res.log == nil {
		res.log = log.New(ioutil.Discard, "", log.LstdFlags)
	}
	if res.network == "" {
		res.network = "tcp"
	}
	if res.raddr == nil {
		res.raddr = laddr.Domain()
	}
	return
}

// The Logger option can be provided to have the connection log debug messages.
func Logger(logger *log.Logger) Option {
	return func(o *options) {
		o.log = logger
	}
}

// The Remote option specifies an endpoint in the XMPP network that we should
// establish the connection to. By default, the domain part of the local
// addresses JID is used.
func Remote(addr *jid.JID) Option {
	return func(o *options) {
		o.raddr = addr
	}
}

// The TLS option fully configures the TLS connection options including the
// certificate chains used, cipher suites, etc.
func TLS(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}

// The SRVExpiration option sets the duration for which the client will cache
// DNS SRV records. The default is 0 (no caching).
func SRVExpiration(exp time.Duration) Option {
	return func(o *options) {
		o.srvExpiration = exp
	}
}

// The Dialer option can be used to configure properties of the connection to
// the XMPP server including the timeout, local address, whether dualstack
// networking is enabled, etc.
func Dialer(dialer net.Dialer) Option {
	return func(o *options) {
		o.dialer = dialer
	}
}

// The network to connect with. Nothing is guaranteed to work if this is not set
// to TCP (the default).
func Network(net string) Option {
	return func(o *options) {
		o.network = net
	}
}
