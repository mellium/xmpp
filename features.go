// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/stream"
)

// A StreamFeature represents a feature that may be selected during stream
// negotiation, eg. STARTTLS, compression, and SASL authentication are all
// stream features.
// Features should be stateless as they may be reused between
// connection attempts, however, a method for passing state between features
// exists on the Parse and Negotiate functions.
type StreamFeature struct {
	// The XML name of the feature in the <stream:feature/> list. If a start
	// element with this name is seen while the connection is reading the features
	// list, it will trigger this StreamFeature's Parse function as a callback.
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
	List func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error)

	// Used to parse the feature that begins with the given xml start element
	// (which should have a Name that matches this stream feature's Name).
	// Returns whether or not the feature is required, and any data that will be
	// needed if the feature is selected for negotiation (eg. the list of
	// mechanisms if the feature was SASL authentication).
	Parse func(ctx context.Context, r xml.TokenReader, start *xml.StartElement) (req bool, data interface{}, err error)

	// A function that will take over the session temporarily while negotiating
	// the feature. The "mask" SessionState represents the state bits that should
	// be flipped after negotiation of the feature is complete. For instance, if
	// this feature creates a security layer (such as TLS) and performs
	// authentication, mask would be set to Authn|Secure, but if it does not
	// authenticate the connection it would just return Secure. If negotiate
	// returns a new io.ReadWriter (probably wrapping the old session.Conn()) the
	// stream will be restarted automatically after Negotiate returns using the
	// new ReadWriter. If this is an initiated connection and the features List
	// call returned a value, that value is passed to the data parameter when
	// Negotiate is called. For instance, in the case of compression this data
	// parameter might be the list of supported algorithms as a slice of strings
	// (or in whatever format the feature implementation has decided upon).
	Negotiate func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, err error)
}

func negotiateFeatures(ctx context.Context, s *Session, features []StreamFeature) (mask SessionState, rw io.ReadWriter, err error) {
	server := (s.state & Received) == Received

	// If we're the server, write the initial stream features.
	var list *streamFeaturesList
	if server {
		list, err = writeStreamFeatures(ctx, s, features)
		if err != nil {
			return mask, nil, err
		}
	}

	var t xml.Token
	var start xml.StartElement
	var ok bool

	if !server {
		// Read a new start stream:features token.
		t, err = s.Token()
		if err != nil {
			return mask, nil, err
		}
		start, ok = t.(xml.StartElement)
		if !ok {
			return mask, nil, stream.BadFormat
		}

		// If we're the client read the rest of the stream features list.
		list, err = readStreamFeatures(ctx, s, start, features)

		switch {
		case err != nil:
			return mask, nil, err
		case list.total == 0:
			// If we received an empty list (or one with no supported features), we're
			// done.
			return Ready, nil, nil
		case len(list.cache) == 0:
			// If we received a list with features we support but where none of them
			// could be negotiated (eg. they were advertised in the wrong order), this
			// is an error:
			// TODO: This error isn't very good.
			return mask, nil, errors.New("xmpp: features advertised out of order")
		}
	}

	var sent bool

	// If the list has any optional items that we support, negotiate them first
	// before moving on to the required items.
	for {
		var data sfData

		if server {
			// Read a new feature to negotiate.
			t, err = s.Token()
			if err != nil {
				return mask, nil, err
			}
			start, ok = t.(xml.StartElement)
			if !ok {
				return mask, nil, stream.BadFormat
			}

			// If the feature was not sent or was already negotiated, error.

			_, negotiated := s.negotiated[start.Name.Space]
			data, sent = list.cache[start.Name.Space]
			if !sent || negotiated {
				// TODO: What should we return here?
				return mask, rw, stream.PolicyViolation
			}
		} else {
			// If we're the client, iterate through the cached features and select one
			// to negotiate.
			for _, v := range list.cache {
				if _, ok := s.negotiated[v.feature.Name.Space]; ok {
					// If this feature has already been negotiated, skip it.
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
				return Ready, nil, nil
			}
		}

		mask, rw, err = data.feature.Negotiate(ctx, s, data.data)
		if err == nil {
			s.state |= mask
		}
		s.negotiated[data.feature.Name.Space] = struct{}{}

		// If we negotiated a required feature or a stream restart is required
		// we're done with this feature set.
		if rw != nil || data.req {
			break
		}
	}

	// If the list contains no required features, negotiation is complete.
	if !list.req {
		mask |= Ready
	}

	return mask, rw, err
}

type sfData struct {
	req     bool
	data    interface{}
	feature StreamFeature
}

type streamFeaturesList struct {
	total int
	req   bool

	// Namespace to sfData
	cache map[string]sfData
}

func getFeature(name xml.Name, features []StreamFeature) (feature StreamFeature, ok bool) {
	for _, f := range features {
		if f.Name == name {
			return f, true
		}
	}
	return feature, false
}

func writeStreamFeatures(ctx context.Context, s *Session, features []StreamFeature) (list *streamFeaturesList, err error) {
	start := xml.StartElement{Name: xml.Name{Space: "", Local: "stream:features"}}
	if err = s.EncodeToken(start); err != nil {
		return
	}

	// Lock the connection features list.
	list = &streamFeaturesList{
		cache: make(map[string]sfData),
	}

	for _, feature := range features {
		// Check if all the necessary bits are set and none of the prohibited bits
		// are set.
		if (s.state&feature.Necessary) == feature.Necessary && (s.state&feature.Prohibited) == 0 {
			var r bool
			r, err = feature.List(ctx, s.out.e, xml.StartElement{
				Name: feature.Name,
			})
			if err != nil {
				return
			}
			list.cache[feature.Name.Space] = sfData{
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
	if err = s.EncodeToken(start.End()); err != nil {
		return
	}
	if err = s.Flush(); err != nil {
		return
	}
	return
}

func readStreamFeatures(ctx context.Context, s *Session, start xml.StartElement, features []StreamFeature) (*streamFeaturesList, error) {
	// TODO: Check if this is a stream error (if so, unmarshal and return).
	switch {
	case start.Name.Local != "features":
		return nil, stream.InvalidXML
	case start.Name.Space != ns.Stream:
		return nil, stream.BadNamespacePrefix
	}

	sf := &streamFeaturesList{
		cache: make(map[string]sfData),
	}

parsefeatures:
	for {
		t, err := s.in.d.Token()
		if err != nil {
			return nil, err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			// If the token is a new feature, see if it's one we handle. If so, parse
			// it. Increment the total features count regardless.
			sf.total++

			// Always add the feature to the list of features, even if we don't
			// support it.
			s.features[tok.Name.Space] = nil

			feature, ok := getFeature(tok.Name, features)

			if ok && s.state&feature.Necessary == feature.Necessary && s.state&feature.Prohibited == 0 {
				req, data, err := feature.Parse(ctx, s.in.d, &tok)
				if err != nil {
					return nil, err
				}

				// TODO: Since we're storing the features data on s.features we can
				// probably remove it from this temporary cache.
				sf.cache[tok.Name.Space] = sfData{
					req:     req,
					data:    data,
					feature: feature,
				}

				// Since we do support the feature, add it to the connections list along
				// with any data returned from Parse.
				s.features[tok.Name.Space] = data
				if req {
					sf.req = true
				}
				continue parsefeatures
			}
			// If the feature is not one we support, skip it.
			if err := xmlstream.Skip(s.in.d); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if tok.Name.Local == "features" && tok.Name.Space == ns.Stream {
				// We've reached the end of the features list!
				return sf, nil
			}
			// Oops, how did that happen? We shouldn't have been able to hit an end
			// element that wasn't the </stream:features> token.
			return nil, stream.InvalidXML
		default:
			return nil, stream.RestrictedXML
		}
	}
}
