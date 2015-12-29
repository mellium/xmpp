// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package client

import (
	"crypto/tls"

	"bitbucket.org/mellium/xmpp/jid"
)

// Option's can be used to configure the client.
type Option func(*options)
type options struct {
	user      *jid.SafeJID
	tlsConfig *tls.Config
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}
	return
}

// The User option sets the username (a bare JID) for which the
func User(j *jid.SafeJID) Option {
	return func(o *options) {
		o.user = j.Bare().(*jid.SafeJID)
	}
}

// The TLS option fully configures the clients TLS connection options including
// the certificate chains used, cipher suites, etc.
func TLS(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}
