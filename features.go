// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
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

	// Used to send the feature in a features list for server connections.
	List func(ctx context.Context, conn *Conn) (req bool, err error)

	// Used to parse the feature that begins with the given xml start element
	// (which should have a Name that matches this stream feature's Name).
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
		_, _, err := writeStreamFeatures(ctx, c)
		if err != nil {
			return err
		}
		panic("Sending stream:features not yet implemented")
	} else {
		t, err := c.in.d.Token()
		if err != nil {
			return err
		}
		start, ok := t.(xml.StartElement)
		if !ok {
			return BadFormat
		}
		list, err := readStreamFeatures(ctx, c, start)
		if err != nil {
			return err
		}

		if list.total == 0 {
			// If we received an empty list, we're done:
			return nil
		}

		// If the list has any required items, negotiate the first required feature.
		// Otherwise just negotiate the first feature in the list.
		var data sfData
		for _, v := range list.cache {
			if !list.req || v.req {
				data = v
				break
			}
		}
		mask, err := data.feature.Negotiate(ctx, c, data.data)
		if err != nil {
			return err
		}
		c.state &= mask

		panic("Receiving stream:features not yet implemented")
	}
}

type sfData struct {
	req     bool
	data    interface{}
	feature StreamFeature
}

type streamFeaturesList struct {
	total int
	req   bool
	cache map[xml.Name]sfData
}

func readStreamFeatures(ctx context.Context, conn *Conn, start xml.StartElement) (*streamFeaturesList, error) {
	switch {
	case start.Name.Local != "features":
		return nil, InvalidXML
	case start.Name.Space != "stream":
		return nil, BadNamespacePrefix
	}

	sf := &streamFeaturesList{}

parsefeatures:
	for {
		t, err := conn.in.d.Token()
		if err != nil {
			return nil, err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			// If the token is a new feature, see if it's one we handle. If so, parse
			// it. Increment the total features count regardless.
			sf.total += 1
			if feature, ok := conn.config.Features[tok.Name]; ok && (conn.state&feature.Necessary) == feature.Necessary && (conn.state&feature.Prohibited) == 0 {
				req, data, err := feature.Parse(ctx, conn, &tok)
				if err == nil {
					return nil, err
				}
				sf.cache[tok.Name] = sfData{
					req:     req,
					data:    data,
					feature: feature,
				}
				if req {
					sf.req = true
				}
				continue parsefeatures
			}
			// If the feature is not one we support, skip it.
			if err := conn.in.d.Skip(); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if tok.Name.Local == "features" && tok.Name.Space == "stream" {
				// We've reached the end of the features list!
				return sf, nil
			}
			// Oops, how did that happen? We shouldn't have been able to hit an end
			// element that wasn't the </stream:features> token.
			return nil, InvalidXML
		default:
			return nil, RestrictedXML
		}
	}

	return sf, nil
}

// Returns the number of stream features written (zero means we've reached the
// end of negotiation), and the number of required features written (zero means
// we've potentially reached the end of negotiation, but the client may
// negotiate more optional features).
func writeStreamFeatures(ctx context.Context, conn *Conn) (n int, req int, err error) {
	if _, err = fmt.Fprint(conn, `<stream:features>`); err != nil {
		return
	}
	for _, feature := range conn.config.Features {
		// Check if all the necessary bits are set and none of the prohibited bits
		// are set.
		if (conn.state&feature.Necessary) == feature.Necessary && (conn.state&feature.Prohibited) == 0 {
			var r bool
			r, err = feature.List(ctx, conn)
			if err != nil {
				return
			}
			if r {
				req += 1
			}
			n += 1
		}
	}
	_, err = fmt.Fprint(conn, `</stream:features>`)
	return
}
