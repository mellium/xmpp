// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
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
	errExpected = errors.New("Expected error")
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

var sendTests = [...]struct {
	r   xml.TokenReader
	err error
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
		r: stanza.Message{To: to, Type: stanza.NormalMessage}.Wrap(nil),
	},
	3: {
		r: stanza.Presence{To: to, Type: stanza.AvailablePresence}.Wrap(nil),
	},
	4: {
		r: stanza.IQ{Type: stanza.ResultIQ}.Wrap(nil),
	},
	5: {
		r: stanza.IQ{Type: stanza.ErrorIQ}.Wrap(nil),
	},
}

func TestSend(t *testing.T) {
	for i, tc := range sendTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
				e := xml.NewEncoder(buf)
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
				t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
			}
			err = s.Close()
			if err != nil {
				t.Errorf("unexpected error closing session: %v", err)
			}
			if tc.err == nil && buf.Len() == 0 {
				t.Errorf("Send wrote no bytes")
			}
		})
	}
}
