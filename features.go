// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmpp/ns"
	"mellium.im/xmpp/streamerror"
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

	// Used to send the feature in a features list for server connections. The
	// start element will have a name that matches the features name and should be
	// used as the outermost tag in the stream (but also may be ignored). List
	// implementations that call e.EncodeToken directly need to call e.Flush when
	// finished to ensure that the XML is written to the underlying writer.
	List func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error)

	// Used to parse the feature that begins with the given xml start element
	// (which should have a Name that matches this stream feature's Name).
	// Returns whether or not the feature is required, and any data that will be
	// needed if the feature is selected for negotiation (eg. the list of
	// mechanisms if the feature was SASL).
	Parse func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (req bool, data interface{}, err error)

	// A function that will take over the session temporarily while negotiating
	// the feature. The "mask" SessionState represents the state bits that should
	// be flipped after negotiation of the feature is complete. For instance, if
	// this feature creates a security layer (such as TLS) and performs
	// authentication, mask would be set to Authn|Secure, but if it does not
	// authenticate the connection it would just return Secure. If negotiate
	// returns a new io.ReadWriteCloser (probably wrapping the old conn.Raw()) the
	// stream will be restarted automatically after Negotiate returns using the
	// new RWC. If this is an initiated connection and the features List call
	// returned a value, that value is passed to the data parameter when Negotiate
	// is called. For instance, in the case of compression this data parameter
	// might be the list of supported algorithms as a slice of strings (or in
	// whatever format the feature implementation has decided upon).
	Negotiate func(ctx context.Context, conn *Conn, data interface{}) (mask SessionState, rwc io.ReadWriteCloser, err error)
}

func (c *Conn) negotiateFeatures(ctx context.Context) (done bool, rwc io.ReadWriteCloser, err error) {
	if (c.state & Received) == Received {
		_, err = writeStreamFeatures(ctx, c)
		if err != nil {
			return false, nil, err
		}
		panic("Sending stream:features not yet implemented")
	}

	t, err := c.in.d.Token()
	if err != nil {
		return done, nil, err
	}
	start, ok := t.(xml.StartElement)
	if !ok {
		return done, nil, streamerror.BadFormat
	}
	list, err := readStreamFeatures(ctx, c, start)

	switch {
	case err != nil:
		return done, nil, err
	case list.total == 0 || len(list.cache) == 0:
		// If we received an empty list (or one with no supported features), we're
		// done.
		return true, nil, nil
	}

	// If the list has any optional items that we support, negotiate them first
	// before moving on to the required items.
	for {
		var data sfData
		for _, v := range list.cache {
			if _, ok := c.negotiated[v.feature.Name]; ok {
				// If this feature has already been negotiated, skip it (servers
				// shouldn't list them in this case, but you never know).
				continue
			}

			// If the feature is optional, select it.
			if !v.req {
				data = v
				break
			}

			// If the feature is required, tentatively select it (but finish looking
			// for optional features).
			if v.req {
				data = v
			}
		}
		// No features that haven't already been negotiated were sent… we're done.
		if data.feature.Name.Local == "" {
			return true, nil, nil
		}
		var mask SessionState
		mask, rwc, err = data.feature.Negotiate(ctx, c, data.data)
		if err == nil {
			c.state |= mask
		}
		c.negotiated[data.feature.Name] = struct{}{}

		// If we negotiated a required feature or a stream restart is required
		// we're done with this feature set.
		if rwc != nil || data.req {
			break
		}
	}

	return !list.req || (c.state&Ready == Ready), rwc, err
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

func writeStreamFeatures(ctx context.Context, conn *Conn) (list *streamFeaturesList, err error) {
	if _, err = fmt.Fprint(conn, `<stream:features>`); err != nil {
		return
	}
	// Lock the connection features list.
	list = &streamFeaturesList{
		cache: make(map[xml.Name]sfData),
	}

	for _, feature := range conn.config.Features {
		// Check if all the necessary bits are set and none of the prohibited bits
		// are set.
		if (conn.state&feature.Necessary) == feature.Necessary && (conn.state&feature.Prohibited) == 0 {
			var r bool
			r, err = feature.List(ctx, conn.out.e, xml.StartElement{
				Name: feature.Name,
			})
			if err != nil {
				return
			}
			list.cache[feature.Name] = sfData{
				req:     r,
				data:    nil,
				feature: feature,
			}
			if r {
				list.req = true
			}
			list.total++
		}
	}
	_, err = fmt.Fprint(conn, `</stream:features>`)
	return
}

func readStreamFeatures(ctx context.Context, conn *Conn, start xml.StartElement) (*streamFeaturesList, error) {
	switch {
	case start.Name.Local != "features":
		return nil, streamerror.InvalidXML
	case start.Name.Space != ns.Stream:
		return nil, streamerror.BadNamespacePrefix
	}

	// Lock the connection features list.
	conn.flock.Lock()
	defer conn.flock.Unlock()

	sf := &streamFeaturesList{
		cache: make(map[xml.Name]sfData),
	}

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
			sf.total++
			conn.features[tok.Name] = struct{}{}
			if feature, ok := conn.config.Features[tok.Name]; ok && (conn.state&feature.Necessary) == feature.Necessary && (conn.state&feature.Prohibited) == 0 {
				req, data, err := feature.Parse(ctx, conn.in.d, &tok)
				if err != nil {
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
			if tok.Name.Local == "features" && tok.Name.Space == ns.Stream {
				// We've reached the end of the features list!
				return sf, nil
			}
			// Oops, how did that happen? We shouldn't have been able to hit an end
			// element that wasn't the </stream:features> token.
			return nil, streamerror.InvalidXML
		default:
			return nil, streamerror.RestrictedXML
		}
	}
}
