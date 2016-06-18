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

// StreamFeatures represents a list of handlers for starting XMPP stream
// features (eg. STARTTLS). While the feature is being negotiated, the given
// function has complete control over the XML stream and the session.
type StreamFeatures map[xml.Name]func(e xml.Encoder, d xml.Decoder)

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

	Features StreamFeatures
}

// NewConfig constructs a new session configuration with some sane defaults. The
// resulting config supports features to auth against most XMPP servers off the
// shelf.
func NewConfig(server, origin *jid.JID) *Config {
	return &Config{
		Location: server,
		Origin:   origin,
		Version:  internal.DefaultVersion,

		Features: StreamFeatures{},
	}
}
