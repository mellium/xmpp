// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
)

// A StreamFeature represents a feature that may be selected during stream
// negotiation. Features should be stateless and usable from multiple goroutines
// unless otherwise specified.
type StreamFeature struct {
	// The XML name of the feature in the <stream:feature/> list. If a start
	// element with this name is seen while the connection is reading the features
	// list, it will trigger this StreamFeature's List function as a callback.
	Name xml.Name

	// Bits that are required before this feature is advertised. For instance, if
	// this feature should only be advertised after the user is authenticated we
	// might set this to "Authn" or if it should be advertised only after the
	// feature is authenticated and encrypted we might set this to "Authn|Secure".
	Necessary SessionState

	// Bits that must be off for this feature to be advertised. For instance, if
	// this feature should only be advertised before the connection is
	// authenticated (eg. if the feature performs authentication itself), we might
	// set this to "Authn".
	Prohibited SessionState

	// Used to send the feature in a features list for server connections, or to
	// decode a feature in a features list for client connections. If this is the
	// initiating side of the connection, start will be the start element that
	// triggered the List call (for use with DecodeElement). List may optionally
	// return data to be passed to the Negotiate function if the feature is
	// selected (eg. a list of mechanisms if the feature is SASL). List should
	// never return an error, but it may panic if it is used on a type of
	// connection which it doesn't support (for example, if a client-only feature
	// is used on a server-side connection). Required means that the value read
	// from (or written too) the stream:fature list indicates that this is a
	// required feature.
	List func(ctx context.Context, conn *Conn) error

	// Used to parse the feature that begins with the given xml start element.
	// Returns whether or not the feature is required, and any data that will be
	// needed if the feature is selected for negotiation (eg. the list of
	// mechanisms if the feature was SASL). If the context expires or is canceled,
	// parse should return the context's error.
	Parse func(ctx context.Context, conn *Conn, start *xml.StartElement) (req bool, data interface{}, err error)

	// A function that will take over the session temporarily while negotiating
	// the feature. The "mask" SessionState represents the state bits that should
	// be flipped after negotiation of the feature is complete. For instance, if
	// this feature creates a security layer (such as TLS) and performs
	// authentication, mask would be set to Authn|Secure|StreamRestartRequired,
	// but if it does not authenticate the connection it would return
	// Secure|StreamRestartRequired. If the mask includes the StreamRestart bit,
	// the stream will be restarted automatically after Negotiate returns (unless
	// it returns an error). If this is an initiated connection and the features
	// List call returned a value, that value is passed to the data parameter when
	// Negotiate is called. For instance, in the case of compression this data
	// parameter might be the list of supported algorithms as a slice of strings
	// (or in whatever format the feature implementation has decided upon).
	Negotiate func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, err error)
}

func (c *Conn) negotiateFeatures(ctx context.Context) error {
	if (c.state & Received) == Received {
		panic("Sending stream:features not yet implemented")
	} else {
		panic("Receiving stream:features not yet implemented")
	}
	return nil
}

// Get a <stream:feature> list. If the next token is not a <stream:feature>
// start element, return an error.
func (c *Conn) parseFeatures() (data map[xml.Name]interface{}, err error) {
	t, err := c.in.d.Token()
	if err != nil {
		return nil, err
	}
	switch tok := t.(type) {
	case xml.StartElement:
		switch {
		case tok.Name.Local != "features":
			return nil, InvalidXML
		case tok.Name.Space != NSStream:
			return nil, BadNamespacePrefix
		}
		// TODO:
		// Start grabbing tokens; for each one, if it's the end features list break,
		// otherwise if the token is a start element loop over the stream features
		// (possibly building a map for quicker lookups on subsequent iterations)
		// and if any of the features have a name that matches the start element,
		// call that features Parse method to consume the rest of the feature before
		// continuing; save any data in a map to be returned and used in the calls
		// to Negotiate.
		panic("Not yet implemented")
	}
	return nil, BadFormat
}
