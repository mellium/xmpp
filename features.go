// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	intstream "mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/stream"
)

const (
	featuresLocal = "features"
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
	// If the stream feature is a legacy feature like resource binding that uses
	// an IQ for negotiation, this should be the name of the IQ payload.
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
	// used as the outermost tag in the stream (but also may be ignored).
	List func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error)

	// Used to parse the feature that begins with the given xml start element
	// (which should have a Name that matches this stream feature's Name).
	// Returns whether or not the feature is required, and any data that will be
	// needed if the feature is selected for negotiation (eg. the list of
	// mechanisms if the feature was SASL authentication).
	Parse func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (req bool, data interface{}, err error)

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

func containsStartTLS(features []StreamFeature) (startTLS StreamFeature, ok bool) {
	for _, feature := range features {
		if feature.Name.Space == ns.StartTLS {
			startTLS, ok = feature, true
			break
		}
	}
	return startTLS, ok
}

func decodeStreamErr(start xml.StartElement, r xml.TokenReader) error {
	if start.Name.Local != "error" || start.Name.Space != stream.NS {
		return nil
	}
	e := stream.Error{}
	err := xml.NewTokenDecoder(r).DecodeElement(&e, &start)
	if err != nil {
		return err
	}
	return e
}

func negotiateFeatures(ctx context.Context, s *Session, first, ws bool, features []StreamFeature) (mask SessionState, rw io.ReadWriter, err error) {
	server := (s.state & Received) == Received

	// If we're the server, write the initial stream features.
	var list *streamFeaturesList
	if server {
		list, err = writeStreamFeatures(ctx, s, ws, features)
		if err != nil {
			return mask, nil, err
		}
	}

	var t xml.Token
	var start xml.StartElement
	var ok bool

	var startTLS StreamFeature
	var doStartTLS bool
	if !server {
		// Read a new start stream:features token.
		t, err = s.in.d.Token()
		if err != nil {
			return mask, nil, err
		}
		start, ok = t.(xml.StartElement)
		if !ok {
			return mask, nil, fmt.Errorf("xmpp: received invalid feature list of type %T", t)
		}
		// Unmarshal any stream errors and return them.
		err = decodeStreamErr(start, s.in.d)
		if err != nil {
			return mask, nil, err
		}

		// If we're the client read the rest of the stream features list.
		list, err = readStreamFeatures(ctx, s, start, features)
		if err != nil {
			return mask, nil, err
		}

		startTLS, doStartTLS = containsStartTLS(features)
		_, advertisedStartTLS := list.cache[ns.StartTLS]
		// If this is the first features list and StartTLS isn't advertised (but
		// is in the features list to be negotiated) and we're not already on a
		// secure connection, try it anyways to prevent downgrade attacks per RFC
		// 7590.
		doStartTLS = first && !advertisedStartTLS && s.State()&Secure != Secure && doStartTLS

		switch {
		case doStartTLS:
			// Skip length checks if we need to negotiate StartTLS for downgrade
			// attack prevention.
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

		oldDecoder := s.in.d
		if server {
			// Read a new feature to negotiate.
			t, err = s.in.d.Token()
			if err != nil {
				return mask, nil, err
			}
			start, ok = t.(xml.StartElement)
			if !ok {
				return mask, nil, fmt.Errorf("xmpp: received invalid start to feature of type %T", t)
			}
			// If this is an IQ (used by legacy features such as resource binding),
			// unwrap it and use the payload's namespace to determine which feature to
			// select.
			// It will be up to the stream feature to actually respond to the IQ
			// correctly.
			var iqStart xml.StartElement
			if isIQ(start.Name) {
				iqStart = start
				t, err = s.in.d.Token()
				if err != nil {
					return mask, nil, err
				}
				start, ok = t.(xml.StartElement)
				if !ok {
					return mask, nil, fmt.Errorf("xmpp: received IQ with invalid payload of type %T", t)
				}
			}

			// If the feature was not sent or was already negotiated, error.
			_, negotiated := s.negotiated[start.Name.Space]
			data, sent = list.cache[start.Name.Space]
			if !sent || negotiated {
				// TODO: What should we return here?
				return mask, rw, stream.PolicyViolation
			}

			// Add the start element(s) that we popped back so that the negotiate
			// function can create a token decoder and have tokens match up and decode
			// things properly.
			if iqStart.Name.Local == "" {
				s.in.d = xmlstream.MultiReader(
					xmlstream.Token(start),
					intstream.Reader(oldDecoder),
				)
			} else {
				s.in.d = xmlstream.MultiReader(
					xmlstream.Token(iqStart),
					xmlstream.Token(start),
					intstream.Reader(oldDecoder),
				)
			}
		} else {
			// If we need to try and negotiate StartTLS even though it wasn't
			// advertised, select it.
			if doStartTLS && startTLS.Name.Space == ns.StartTLS {
				data = sfData{
					req:     true,
					feature: startTLS,
				}
			} else {
				// If we're the client, iterate through the cached features and select
				// one to negotiate.
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

					// If the feature is required, tentatively select it (but finish
					// looking for optional features).
					data = v
				}
			}

			// No features that haven't already been negotiated were sent… we're done.
			if data.feature.Name.Local == "" {
				return Ready, nil, nil
			}
			s.in.d = intstream.Reader(oldDecoder)
		}

		mask, rw, err = data.feature.Negotiate(ctx, s, s.features[data.feature.Name.Space])
		s.in.d = oldDecoder
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

	// If the list contains no required features and a stream restart is not
	// required,  negotiation is complete.
	if !list.req {
		mask |= Ready
	}

	return mask, rw, err
}

type sfData struct {
	req     bool
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

func writeStreamFeatures(ctx context.Context, s *Session, ws bool, features []StreamFeature) (list *streamFeaturesList, err error) {
	var start xml.StartElement
	if ws {
		// There is no wrapping "stream:stream" element in the websocket
		// subprotocol, so set the namespace using the unprefixed form.
		start.Name = xml.Name{Space: stream.NS, Local: "features"}
	} else {
		start.Name = xml.Name{Local: "stream:features"}
	}
	w := s.TokenWriter()
	defer w.Close()
	if err = w.EncodeToken(start); err != nil {
		return
	}

	// Lock the connection features list.
	list = &streamFeaturesList{
		cache: make(map[string]sfData),
	}

	for _, feature := range features {
		// Check if all the necessary bits are set and none of the prohibited bits
		// are set.
		if (s.state&feature.Necessary) == feature.Necessary &&
			(s.state&feature.Prohibited) == 0 {
			var r bool
			r, err = feature.List(ctx, s.out.e, xml.StartElement{
				Name: feature.Name,
			})
			if err != nil {
				return list, err
			}
			list.cache[feature.Name.Space] = sfData{
				req:     r,
				feature: feature,
			}
			if r {
				list.req = true
			}
			list.total++
		}
	}
	if err = w.EncodeToken(start.End()); err != nil {
		return list, err
	}
	if err = w.Flush(); err != nil {
		return list, err
	}
	return list, err
}

func nextElementDecoder(r xml.TokenReader, start xml.StartElement) *xml.Decoder {
	d := xml.NewTokenDecoder(xmlstream.MultiReader(
		xmlstream.Token(start),
		xmlstream.InnerElement(r),
	))
	// This isn't ideal, but we have to provide the start element to the decoder
	// and then pop it and ignore it (since it had already been read fro mthe
	// underlying reader) to setup the internal state of the new decoder.
	// This was the only way I could contrive to provide Parse calls with a
	// Decoder that won't error when it reaches the end token.

	/* #nosec */
	d.Token()
	return d
}

func readStreamFeatures(ctx context.Context, s *Session, start xml.StartElement, features []StreamFeature) (*streamFeaturesList, error) {
	switch {
	case start.Name.Local != featuresLocal:
		return nil, stream.InvalidXML
	case start.Name.Space != stream.NS:
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
			limitDecoder := nextElementDecoder(s.in.d, tok)

			// If the token is a new feature, see if it's one we handle. If so, parse
			// it. Increment the total features count regardless.
			sf.total++

			// Always add the feature to the list of features, even if we don't
			// support it, it just won't contain any parse output.
			s.features[tok.Name.Space] = nil

			feature, ok := getFeature(tok.Name, features)
			if ok {
				req, data, err := feature.Parse(ctx, limitDecoder, &tok)
				if err != nil {
					return nil, err
				}
				sf.req = sf.req || req

				if s.state&feature.Necessary == feature.Necessary &&
					s.state&feature.Prohibited == 0 {

					sf.cache[tok.Name.Space] = sfData{
						req:     req,
						feature: feature,
					}

					// Since we do support the feature, add it to the connections list
					// along with any data returned from Parse.
					s.features[tok.Name.Space] = data
					continue parsefeatures
				}
			}
			// Advance to the end of the feature element (in case the parse function
			// didn't consume the entire feature or we did not support the feature and
			// need to skip it).
			_, err := xmlstream.Copy(xmlstream.Discard(), limitDecoder)
			if err != nil {
				return nil, err
			}
		case xml.EndElement:
			if tok.Name.Local == featuresLocal && tok.Name.Space == stream.NS {
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
