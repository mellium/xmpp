// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"crypto/tls"
	"encoding/xml"

	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
)

// Config represents the configuration of an XMPP session.
type Config struct {
	// An XMPP server address.
	Location *jid.JID

	// An XMPP connection origin (local address).
	Origin *jid.JID

	// XMPP protocol version
	Version internal.Version

	// TLS config for STARTTLS.
	TLSConfig *tls.Config

	// True if this is a server-to-server session.
	S2S bool

	// The supported stream features.
	Features map[xml.Name]StreamFeature
}

// NewClientConfig constructs a new client-to-server session configuration with
// sane defaults.
func NewClientConfig(origin *jid.JID) *Config {
	return &Config{
		Location: origin.Domain(),
		Origin:   origin,
		Version:  internal.DefaultVersion,

		Features: map[xml.Name]StreamFeature{
		// TODO
		},
	}
}

// NewServerConfig constructs a new server-to-server session configuration with
// sane defaults.
func NewServerConfig(location, origin *jid.JID) *Config {
	return &Config{
		Location: location,
		Origin:   origin,
		S2S:      true,
		Version:  internal.DefaultVersion,

		Features: map[xml.Name]StreamFeature{
		// TODO
		},
	}
}
