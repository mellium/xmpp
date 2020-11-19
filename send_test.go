// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
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

type toksReader []xml.Token

func (r *toksReader) Token() (xml.Token, error) {
	if len(*r) == 0 {
		return nil, io.EOF
	}

	var t xml.Token
	t, *r = (*r)[0], (*r)[1:]
	return t, nil
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
				s := xmpptest.NewClientServer(0, xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				resp, err := s.SendIQElement(ctx, tc.payload, tc.iq)
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
				s := xmpptest.NewClientServer(0, xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				resp, err := s.SendIQ(ctx, tc.iq.Wrap(tc.payload))
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
		s := xmpptest.NewClientServer(0, xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(t, stanza.IQ{ID: testIQID, Type: stanza.ResultIQ}.Wrap(nil))
			return err
		}))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := s.EncodeIQElement(ctx, struct {
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
		s := xmpptest.NewClientServer(0, xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			_, err := xmlstream.Copy(t, stanza.IQ{ID: testIQID, Type: stanza.ResultIQ}.Wrap(nil))
			return err
		}))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := s.EncodeIQ(ctx, struct {
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
	r          xml.TokenReader
	err        error
	resp       xml.TokenReader
	writesBody bool
}{
	0: {
		r:   errReader{err: errExpected},
		err: errExpected,
	},
	1: {
		r: &toksReader{
			xml.EndElement{Name: xml.Name{Local: "iq"}},
		},
		err: xmpp.ErrNotStart,
	},
	2: {
		r:          stanza.Message{To: to, Type: stanza.NormalMessage}.Wrap(nil),
		writesBody: true,
	},
	3: {
		r:          stanza.Presence{To: to, Type: stanza.AvailablePresence}.Wrap(nil),
		writesBody: true,
	},
	4: {
		r:          stanza.IQ{Type: stanza.ResultIQ}.Wrap(nil),
		writesBody: true,
	},
	5: {
		r:          stanza.IQ{Type: stanza.ErrorIQ}.Wrap(nil),
		writesBody: true,
	},
}

func TestSend(t *testing.T) {
	for i, tc := range sendTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			br := &bytes.Buffer{}
			bw := &bytes.Buffer{}
			s := xmpptest.NewSession(0, struct {
				io.Reader
				io.Writer
			}{
				Reader: br,
				Writer: bw,
			})
			defer func() {
				if err := s.Close(); err != nil {
					t.Errorf("Error closing session: %q", err)
				}
			}()
			if tc.resp != nil {
				e := xml.NewEncoder(br)
				_, err := xmlstream.Copy(e, tc.resp)
				if err != nil {
					t.Logf("error responding: %q", err)
				}
				err = e.Flush()
				if err != nil {
					t.Logf("error flushing after responding: %q", err)
				}
			}

			go func() {
				err := s.Serve(nil)
				if err != nil && err != io.EOF {
					panic(err)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := s.Send(ctx, tc.r)
			if err != tc.err {
				t.Errorf("Unexpected error, want=%q, got=%q", tc.err, err)
			}
			if empty := bw.Len() != 0; tc.writesBody != empty {
				t.Errorf("Unexpected body, want=%t, got=%t", tc.writesBody, empty)
			}
		})
	}
}
