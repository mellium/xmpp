// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package client

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"time"
)

// Option's can be used to configure the client.
type Option func(*options)
type options struct {
	log           *log.Logger
	tlsConfig     *tls.Config
	srvExpiration time.Duration
	dialer        net.Dialer
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}

	// Log to /dev/null by default.
	if res.log == nil {
		res.log = log.New(ioutil.Discard, "", log.LstdFlags)
	}
	return
}

// The Logger option can be provided to have Client log debug messages and other
// helpful info.
func Logger(logger *log.Logger) Option {
	return func(o *options) {
		o.log = logger
	}
}

// The TLS option fully configures the clients TLS connection options including
// the certificate chains used, cipher suites, etc.
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

// ConnTimeout sets a timeout on connection attempts to the server (not
// including SRV lookup time, for which the timeout is set by the system). Some
// systems may override long timeouts and break the connection earlier.
func ConnTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.dialer.Timeout = timeout
	}
}

// Deadline is the absolute point in time after which connection attempts to the
// server will fail. If Timeout is set, it may fail earlier. Zero means no
// deadline, or dependent on the operating system as with the ConnTimeout
// option.
func Deadline(deadline time.Time) Option {
	return func(o *options) {
		o.dialer.Deadline = deadline
	}
}

// LocalAddr is the local address to use when connecting to the server. The
// address must be of a compatible type for a TCP connection. If nil (the
// default), a local address is automatically chosen.
func LocalAddr(addr net.Addr) Option {
	return func(o *options) {
		o.dialer.LocalAddr = addr
	}
}

var (
	// DualStack enables RFC 6555-compliant "Happy Eyeballs" dialing when the
	// destination is a host name with both IPv4 and IPv6 addresses. This allows a
	// client to tolerate networks where one address family is silently broken.
	DualStack Option = dualstack
)

var dualstack = func(o *options) {
	o.dialer.DualStack = true
}

// FallbackDelay specifies the length of time to wait before spawning a fallback
// connection when DualStack is enabled. If zero, a default delay of 300ms is
// used.
func FallbackDelay(delay time.Duration) Option {
	return func(o *options) {
		o.dialer.FallbackDelay = delay
	}
}
