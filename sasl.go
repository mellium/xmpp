// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/internal/saslerr"
)

var (
	errNoMechanisms      = errors.New(`xmpp: no matching SASL mechanisms found`)
	errUnexpectedPayload = errors.New(`xmpp: unexpected payload encountered during auth`)
	errTerminated        = errors.New(`xmpp: the remote entity terminated authentication`)
)

// SASL returns a stream feature for performing authentication using the Simple
// Authentication and Security Layer (SASL) as defined in RFC 4422.
// It panics if no mechanisms are specified.
// The order in which mechanisms are specified will be the preferred order, so
// stronger mechanisms should be listed first.
//
// Identity is used when a user wants to act on behalf of another user.
// For instance, an admin might want to log in as another user to help them
// troubleshoot an issue.
// Normally it is left blank and the localpart of the Origin JID is used.
func SASL(identity, password string, mechanisms ...sasl.Mechanism) StreamFeature {
	return newSASL(identity, password, nil, mechanisms...)
}

// SASLServer is like SASL but the returned feature uses the provided
// permissions func to validate credentials provided by the client.
func SASLServer(permissions func(*sasl.Negotiator) bool, mechanisms ...sasl.Mechanism) StreamFeature {
	return newSASL("", "", permissions, mechanisms...)
}

func newSASL(identity, password string, permissions func(*sasl.Negotiator) bool, mechanisms ...sasl.Mechanism) StreamFeature {
	if len(mechanisms) == 0 {
		panic("xmpp: must specify at least one SASL mechanism")
	}
	return StreamFeature{
		Name:       xml.Name{Space: ns.SASL, Local: "mechanisms"},
		Necessary:  Secure,
		Prohibited: Authn,
		List: func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (bool, error) {
			err := e.EncodeToken(start)
			if err != nil {
				return true, err
			}

			startMechanism := xml.StartElement{Name: xml.Name{Space: "", Local: "mechanism"}}
			for _, m := range mechanisms {
				select {
				case <-ctx.Done():
					return true, ctx.Err()
				default:
				}

				if err = e.EncodeToken(startMechanism); err != nil {
					return true, err
				}
				if err = e.EncodeToken(xml.CharData(m.Name)); err != nil {
					return true, err
				}
				if err = e.EncodeToken(startMechanism.End()); err != nil {
					return true, err
				}
			}
			return true, e.EncodeToken(start.End())
		},
		Parse: func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
				List    []string `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanism"`
			}{}
			err := d.DecodeElement(&parsed, start)
			return true, parsed.List, err
		},
		Negotiate: func(ctx context.Context, session *Session, data interface{}) (SessionState, io.ReadWriter, error) {
			if (session.State() & Received) == Received {
				return negotiateServer(ctx, identity, password, permissions, session, data, mechanisms...)
			}

			return negotiateClient(ctx, identity, password, session, data, mechanisms...)
		},
	}
}

func negotiateServer(ctx context.Context, identity, password string, permissions func(*sasl.Negotiator) bool, session *Session, data interface{}, mechanisms ...sasl.Mechanism) (SessionState, io.ReadWriter, error) {
	w := session.TokenWriter()
	/* #nosec */
	defer w.Close()
	r := session.TokenReader()
	/* #nosec */
	defer r.Close()
	d := xml.NewTokenDecoder(r)

	var (
		selected sasl.Mechanism
		server   *sasl.Negotiator
		resp     []byte
	)
	for more := true; more; {
		tok, err := d.Token()
		if err != nil {
			return 0, nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			return 0, nil, errUnexpectedPayload
		}
		fail, ok, err := decodeIfSASLErr(d, start)
		switch {
		case err != nil:
			return 0, nil, err
		case ok:
			return 0, nil, fail
		}
		selection := struct {
			XMLName xml.Name
			Name    string `xml:"mechanism,attr"`
			Payload []byte `xml:",chardata"`
		}{}
		err = d.DecodeElement(&selection, &start)
		if err != nil {
			return 0, nil, err
		}

		switch selection.XMLName {
		case xml.Name{Space: ns.SASL, Local: "auth"}:
			selected = sasl.Mechanism{}
			for _, m := range mechanisms {
				if selection.Name == m.Name {
					selected = m
					break
				}
			}

			// No matching mechanism found…
			if selected.Name == "" {
				err = sendSASLError(w, saslerr.Failure{
					Condition: saslerr.InvalidMechanism,
				})
				if err != nil {
					return 0, nil, err
				}
				return 0, nil, errNoMechanisms
			}

			opts := []sasl.Option{
				sasl.Credentials(func() ([]byte, []byte, []byte) {
					return []byte(session.LocalAddr().Localpart()), []byte(password), []byte(identity)
				}),
			}

			if connState := session.ConnectionState(); connState.Version != 0 {
				opts = append(opts, sasl.TLSState(connState))
			}

			server = sasl.NewServer(selected, permissions, opts...)
		case xml.Name{Space: ns.SASL, Local: "abort"}:
			err = sendSASLError(w, saslerr.Failure{
				Condition: saslerr.Aborted,
			})
			if err != nil {
				return 0, nil, err
			}
			return 0, nil, errTerminated
		case xml.Name{Space: ns.SASL, Local: "response"}:
			// We never got the initial <auth/> payload and selected a mechanism. This
			// would be bad, so error out.
			if server == nil || selected.Name == "" {
				err = sendSASLError(w, saslerr.Failure{
					Condition: saslerr.MalformedRequest,
				})
				if err != nil {
					return 0, nil, err
				}
				return 0, nil, errUnexpectedPayload
			}
		default:
			err = sendSASLError(w, saslerr.Failure{
				Condition: saslerr.MalformedRequest,
			})
			if err != nil {
				return 0, nil, err
			}
			return 0, nil, errUnexpectedPayload
		}

		// An empty payload or a payload of "=" (the correct way to transmit an empty
		// payload) will result in a zero length buffer).
		l := base64.StdEncoding.DecodedLen(len(selection.Payload))
		var decodedData []byte
		if l > 1 {
			decodedData = make([]byte, l)
			_, err = base64.StdEncoding.Decode(decodedData, selection.Payload)
			if err != nil {
				return 0, nil, err
			}
		}
		more, resp, err = server.Step(decodedData)
		switch err {
		case nil:
		case sasl.ErrAuthn:
			e := sendSASLError(w, saslerr.Failure{
				Condition: saslerr.NotAuthorized,
			})
			if e != nil {
				err = e
			}
			return 0, nil, err
		default:
			return 0, nil, err
		}

		// RFC6120 §6.4.2:
		//     If the initiating entity needs to send a zero-length initial
		//     response, it MUST transmit the response as a single equals sign
		//     character ("="), which indicates that the response is present but
		//     contains no data.
		if more {
			var encodedResp []byte
			if len(resp) == 0 {
				encodedResp = []byte{'='}
			} else {
				encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
				base64.StdEncoding.Encode(encodedResp, resp)
			}

			_, err = xmlstream.Copy(w, xmlstream.Wrap(
				xmlstream.Token(xml.CharData(encodedResp)),
				xml.StartElement{
					Name: xml.Name{Space: ns.SASL, Local: "challenge"},
				},
			))
			if err != nil {
				return 0, nil, err
			}
			err = w.Flush()
			if err != nil {
				return 0, nil, err
			}
		}
	}

	// If there is no more, but there was no error, auth was successful!
	var encodedResp []byte
	if len(resp) >= 0 {
		encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
		base64.StdEncoding.Encode(encodedResp, resp)
		_, err := xmlstream.Copy(w, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(encodedResp)),
			xml.StartElement{
				Name: xml.Name{Space: ns.SASL, Local: "success"},
			},
		))
		if err != nil {
			return 0, nil, err
		}
		return Authn, session.Conn(), nil
	}

	_, err := xmlstream.Copy(w, xmlstream.Wrap(
		nil,
		xml.StartElement{
			Name: xml.Name{Space: ns.SASL, Local: "success"},
		},
	))
	if err != nil {
		return 0, nil, err
	}
	return Authn, session.Conn(), nil
}

func negotiateClient(ctx context.Context, identity, password string, session *Session, data interface{}, mechanisms ...sasl.Mechanism) (SessionState, io.ReadWriter, error) {
	var mask SessionState
	w := session.TokenWriter()
	/* #nosec */
	defer w.Close()

	var selected sasl.Mechanism
	// Select a mechanism, preferring the client order.
selectmechanism:
	for _, m := range mechanisms {
		for _, name := range data.([]string) {
			if name == m.Name {
				selected = m
				break selectmechanism
			}
		}
	}
	// No matching mechanism found…
	if selected.Name == "" {
		return mask, nil, errNoMechanisms
	}

	opts := []sasl.Option{
		sasl.Credentials(func() ([]byte, []byte, []byte) {
			return []byte(session.LocalAddr().Localpart()), []byte(password), []byte(identity)
		}),
		sasl.RemoteMechanisms(data.([]string)...),
	}

	if connState := session.ConnectionState(); connState.Version != 0 {
		opts = append(opts, sasl.TLSState(connState))
	}

	client := sasl.NewClient(selected, opts...)
	more, resp, err := client.Step(nil)
	if err != nil {
		return mask, nil, err
	}

	// RFC6120 §6.4.2:
	//     If the initiating entity needs to send a zero-length initial
	//     response, it MUST transmit the response as a single equals sign
	//     character ("="), which indicates that the response is present but
	//     contains no data.
	var encodedResp []byte
	if len(resp) == 0 {
		encodedResp = []byte{'='}
	} else {
		encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
		base64.StdEncoding.Encode(encodedResp, resp)
	}

	// Send <auth/> and the initial payload to start SASL auth.
	_, err = xmlstream.Copy(w, xmlstream.Wrap(
		xmlstream.Token(xml.CharData(encodedResp)),
		xml.StartElement{
			Name: xml.Name{Space: ns.SASL, Local: "auth"},
			Attr: []xml.Attr{{
				Name:  xml.Name{Local: "mechanism"},
				Value: selected.Name,
			}},
		},
	))
	if err != nil {
		return mask, nil, err
	}
	err = w.Flush()
	if err != nil {
		return mask, nil, err
	}

	r := session.TokenReader()
	defer r.Close()
	d := xml.NewTokenDecoder(r)

	// If we're already done after the first step, decode the <success/> or
	// <failure/> before we exit.
	if !more {
		tok, err := d.Token()
		if err != nil {
			return mask, nil, err
		}
		if t, ok := tok.(xml.StartElement); ok {
			// TODO: Handle the additional data that could be returned if
			// success?
			_, _, err := decodeSASLChallenge(d, t, false)
			if err != nil {
				return mask, nil, err
			}
		} else {
			return mask, nil, errUnexpectedPayload
		}
	}

	success := false
	for more {
		select {
		case <-ctx.Done():
			return mask, nil, ctx.Err()
		default:
		}
		tok, err := d.Token()
		if err != nil {
			return mask, nil, err
		}
		var challenge []byte
		if t, ok := tok.(xml.StartElement); ok {
			challenge, success, err = decodeSASLChallenge(d, t, true)
			if err != nil {
				return mask, nil, err
			}
		} else {
			return mask, nil, errUnexpectedPayload
		}
		if more, resp, err = client.Step(challenge); err != nil {
			return mask, nil, err
		}
		if !more && success {
			// We're done with SASL and we're successful
			break
		}

		var encodedResp []byte
		if len(resp) == 0 {
			encodedResp = []byte{'='}
		} else {
			encodedResp = make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
			base64.StdEncoding.Encode(encodedResp, resp)
		}

		// TODO: What happens if there's more and success (broken server)?
		_, err = xmlstream.Copy(w, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(encodedResp)),
			xml.StartElement{
				Name: xml.Name{Space: ns.SASL, Local: "response"},
			},
		))
		if err != nil {
			return mask, nil, err
		}
		err = w.Flush()
		if err != nil {
			return mask, nil, err
		}
	}
	return Authn, session.Conn(), nil
}

func decodeSASLChallenge(d *xml.Decoder, start xml.StartElement, allowChallenge bool) (challenge []byte, success bool, err error) {
	switch start.Name {
	case xml.Name{Space: ns.SASL, Local: "challenge"}, xml.Name{Space: ns.SASL, Local: "success"}:
		if !allowChallenge && start.Name.Local == "challenge" {
			return nil, false, errUnexpectedPayload
		}
		challenge := struct {
			Data []byte `xml:",chardata"`
		}{}
		if err = d.DecodeElement(&challenge, &start); err != nil {
			return nil, false, err
		}

		decodedChallenge := make([]byte, base64.StdEncoding.DecodedLen(len(challenge.Data)))
		n, err := base64.StdEncoding.Decode(decodedChallenge, challenge.Data)
		if err != nil {
			return nil, false, err
		}
		decodedChallenge = decodedChallenge[:n]

		return decodedChallenge, start.Name.Local == "success", nil
	case xml.Name{Space: ns.SASL, Local: "failure"}:
		fail := saslerr.Failure{}
		if err = d.DecodeElement(&fail, &start); err != nil {
			return nil, false, err
		}
		return nil, false, fail
	default:
		return nil, false, errUnexpectedPayload
	}
}

func sendSASLError(w xmlstream.TokenWriteFlusher, fail saslerr.Failure) error {
	_, err := xmlstream.Copy(w, fail.TokenReader())
	if err != nil {
		return err
	}
	return w.Flush()
}

func decodeIfSASLErr(d *xml.Decoder, start xml.StartElement) (saslerr.Failure, bool, error) {
	if start.Name.Local != "failure" || start.Name.Space != ns.SASL {
		return saslerr.Failure{}, false, nil
	}

	fail := saslerr.Failure{}
	return fail, true, d.DecodeElement(&fail, &start)
}
