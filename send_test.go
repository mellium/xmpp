// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"strconv"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	testIQID = "123"
)

type errReader struct{ err error }

func (r errReader) Token() (xml.Token, error) {
	return nil, r.err
}

var (
	errExpected = errors.New("expected error")
	to          = jid.MustParse("test@example.net")
)

var sendIQTests = [...]struct {
	iq         stanza.IQ
	payload    xml.TokenReader
	err        error
	writesBody bool
	resp       *stanza.IQ
}{
	0: {
		iq:         stanza.IQ{ID: testIQID, Type: stanza.GetIQ},
		writesBody: true,
		resp:       &stanza.IQ{ID: testIQID, Type: stanza.ResultIQ},
	},
	1: {
		iq:         stanza.IQ{ID: testIQID, Type: stanza.SetIQ},
		writesBody: true,
		resp:       &stanza.IQ{ID: testIQID, Type: stanza.ErrorIQ},
	},
	2: {
		iq:         stanza.IQ{Type: stanza.ResultIQ, ID: testIQID},
		writesBody: true,
	},
	3: {
		iq:         stanza.IQ{Type: stanza.ErrorIQ, ID: testIQID},
		writesBody: true,
	},
}

func TestSendIQ(t *testing.T) {
	for i, tc := range sendIQTests {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("SendIQElement", func(t *testing.T) {
				s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				resp, err := s.Client.SendIQElement(ctx, tc.payload, tc.iq)
				if err != tc.err {
					t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
				}
				respIQ := stanza.IQ{}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("Error closing response: %q", err)
						}
					}()
					err = xml.NewTokenDecoder(resp).Decode(&respIQ)
					if err != nil {
						t.Errorf("error decoding response: %v", err)
					}
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("Expected response, but got none")
				case resp != nil && tc.resp == nil:
					t.Errorf("Did not expect response, but got: %+v", respIQ)
				}
			})
			t.Run("SendIQ", func(t *testing.T) {
				s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				resp, err := s.Client.SendIQ(ctx, tc.iq.Wrap(tc.payload))
				if err != tc.err {
					t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
				}
				respIQ := stanza.IQ{}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("Error closing response: %q", err)
						}
					}()
					err = xml.NewTokenDecoder(resp).Decode(&respIQ)
					if err != nil {
						t.Errorf("error decoding response: %v", err)
					}
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("Expected response, but got none")
				case resp != nil && tc.resp == nil:
					t.Errorf("Did not expect response, but got: %+v", respIQ)
				}
			})
		})
	}
}

func TestEncodeIQ(t *testing.T) {
	t.Run("EncodeIQElement", func(t *testing.T) {
		s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(t, stanza.IQ{ID: testIQID, Type: stanza.ResultIQ}.Wrap(nil))
			return err
		}))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := s.Client.EncodeIQElement(ctx, struct {
			XMLName xml.Name `xml:"urn:xmpp:time time"`
		}{}, stanza.IQ{
			ID:   testIQID,
			Type: stanza.GetIQ,
		})
		if err != nil {
			t.Errorf("Unexpected error %q", err)
		}
		if resp != nil {
			defer func() {
				if err := resp.Close(); err != nil {
					t.Errorf("Error closing response: %q", err)
				}
			}()
		}
		if resp == nil {
			t.Errorf("Expected response, but got none")
		}
	})
	t.Run("EncodeIQ", func(t *testing.T) {
		s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(t, stanza.IQ{ID: testIQID, Type: stanza.ResultIQ}.Wrap(nil))
			return err
		}))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := s.Client.EncodeIQ(ctx, struct {
			stanza.IQ

			Payload struct{} `xml:"urn:xmpp:time time"`
		}{
			IQ: stanza.IQ{
				ID:   testIQID,
				Type: stanza.GetIQ,
			},
		})
		if err != nil {
			t.Errorf("Got unexpected error encoding: %v", err)
		}
		if resp != nil {
			defer func() {
				if err := resp.Close(); err != nil {
					t.Errorf("Error closing response: %q", err)
				}
			}()
		}
		if resp == nil {
			t.Errorf("Expected response, but got none")
		}
	})
}

// zeroID will be the ID of stanzas that have an empty ID in the send tests.
// Normally a random ID is generated, but for the purposes of the tests the
// source of randomness has been replaced with a reader that only reads zeros.
const zeroID = "0000000000000000"

var sendTests = [...]struct {
	r   xml.TokenReader
	err error
	out string
}{
	0: {
		r:   errReader{err: errExpected},
		err: errExpected,
	},
	1: {
		r: &xmpptest.Tokens{
			xml.EndElement{Name: xml.Name{Local: "iq"}},
		},
		err: xmpp.ErrNotStart,
	},
	2: {
		r:   stanza.Message{To: to, Type: stanza.NormalMessage}.Wrap(nil),
		out: `<message xmlns="jabber:client" type="normal" to="test@example.net" id="` + zeroID + `"></message>`,
	},
	3: {
		r:   stanza.Presence{To: to, Type: stanza.AvailablePresence}.Wrap(nil),
		out: `<presence xmlns="jabber:client" to="test@example.net" id="` + zeroID + `"></presence>`,
	},
	4: {
		r:   stanza.Presence{To: to, Type: stanza.SubscribePresence, ID: "123"}.Wrap(nil),
		out: `<presence xmlns="jabber:client" type="subscribe" to="test@example.net" id="123"></presence>`,
	},
	5: {
		r: stanza.IQ{Type: stanza.ResultIQ}.Wrap(nil),
	},
	6: {
		r: stanza.IQ{Type: stanza.ErrorIQ}.Wrap(nil),
	},
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestSend(t *testing.T) {
	// For this test (and this test only) override the global source of randomness
	// so that we can deterministically test the output of stanzas even if a
	// random ID would be generated.
	origRand := rand.Reader
	rand.Reader = zeroReader{}
	defer func() {
		rand.Reader = origRand
	}()

	for i, tc := range sendTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
				e := xml.NewEncoder(buf)
				// Filter out xmlns before re-encoding to avoid the buggy Go XLM encoder
				// which will gladly duplicate xmlns attributes if namespace is set in
				// the Name field and in the attributes.
				filtered := start.Attr[:0]
				for _, attr := range start.Attr {
					if attr.Name.Local != "xmlns" {
						filtered = append(filtered, attr)
					}
				}
				start.Attr = filtered
				err := e.EncodeToken(*start)
				if err != nil {
					return err
				}
				_, err = xmlstream.Copy(e, t)
				if err != nil {
					return err
				}
				return e.Flush()
			}))

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := s.Client.Send(ctx, tc.r)
			if err != tc.err {
				t.Errorf("unexpected error, want=%q, got=%q", tc.err, err)
			}
			err = s.Close()
			if err != nil {
				t.Errorf("unexpected error closing session: %v", err)
			}
			if tc.err == nil && buf.Len() == 0 {
				t.Errorf("send wrote no bytes")
			}
			if s := buf.String(); tc.out != "" && tc.out != s {
				t.Errorf("got wrong output:\nwant=%s, \ngot=%s", tc.out, s)
			}
		})
	}
}
