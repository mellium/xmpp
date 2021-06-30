// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"context"
	"encoding/xml"
	"strconv"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/stanza"
)

var sendPresenceTests = [...]struct {
	msg        stanza.Presence
	canceled   bool
	payload    xml.TokenReader
	err        error
	writesBody bool
	resp       *stanza.Presence
}{
	0: {
		msg:        stanza.Presence{ID: testIQID, Type: stanza.AvailablePresence},
		writesBody: true,
		resp:       &stanza.Presence{ID: testIQID, Type: stanza.ErrorPresence},
	},
	1: {
		msg:        stanza.Presence{ID: testIQID, Type: stanza.ProbePresence},
		writesBody: true,
		resp:       &stanza.Presence{ID: testIQID, Type: stanza.ErrorPresence},
	},
	2: {
		msg:        stanza.Presence{ID: testIQID, Type: stanza.ErrorPresence},
		writesBody: true,
	},
	3: {
		msg:        stanza.Presence{ID: testIQID, Type: stanza.SubscribePresence},
		canceled:   true,
		writesBody: true,
		err:        context.Canceled,
	},
}

func TestSendPresence(t *testing.T) {
	for i, tc := range sendPresenceTests {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("SendPresenceElement", func(t *testing.T) {
				s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if tc.canceled {
					cancel()
				} else {
					defer cancel()
				}

				resp, err := s.Client.SendPresenceElement(ctx, tc.payload, tc.msg)
				if err != tc.err {
					t.Errorf("unexpected error, want=%q, got=%q", tc.err, err)
				}
				respMsg := stanza.Presence{}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("Error closing response: %q", err)
						}
					}()
					err = xml.NewTokenDecoder(resp).Decode(&respMsg)
					if err != nil {
						t.Errorf("error decoding response: %v", err)
					}
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("expected response, but got none")
				case resp != nil && tc.resp == nil:
					t.Errorf("did not expect response, but got: %+v", respMsg)
				}
			})
			t.Run("SendPresence", func(t *testing.T) {
				s := xmpptest.NewClientServer(xmpptest.ServerHandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
					if tc.resp != nil {
						_, err := xmlstream.Copy(t, tc.resp.Wrap(nil))
						return err
					}
					return nil
				}))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if tc.canceled {
					cancel()
				} else {
					defer cancel()
				}

				resp, err := s.Client.SendPresence(ctx, tc.msg.Wrap(tc.payload))
				if err != tc.err {
					t.Errorf("unexpected error, want=%q, got=%q", tc.err, err)
				}
				respMsg := stanza.Presence{}
				if resp != nil {
					defer func() {
						if err := resp.Close(); err != nil {
							t.Errorf("error closing response: %q", err)
						}
					}()
					err = xml.NewTokenDecoder(resp).Decode(&respMsg)
					if err != nil {
						t.Errorf("error decoding response: %v", err)
					}
				}
				switch {
				case resp == nil && tc.resp != nil:
					t.Errorf("expected response, but got none")
				case resp != nil && tc.resp == nil:
					t.Errorf("did not expect response, but got: %+v", respMsg)
				}
			})
		})
	}
}
