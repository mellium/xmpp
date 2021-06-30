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

var sendMessageTests = [...]struct {
	msg        stanza.Message
	canceled   bool
	payload    xml.TokenReader
	err        error
	writesBody bool
	resp       *stanza.Message
}{
	0: {
		msg:        stanza.Message{ID: testIQID, Type: stanza.NormalMessage},
		writesBody: true,
		resp:       &stanza.Message{ID: testIQID, Type: stanza.ErrorMessage},
	},
	1: {
		msg:        stanza.Message{ID: testIQID, Type: stanza.GroupChatMessage},
		writesBody: true,
		resp:       &stanza.Message{ID: testIQID, Type: stanza.ErrorMessage},
	},
	2: {
		msg:        stanza.Message{Type: stanza.ErrorMessage, ID: testIQID},
		writesBody: true,
	},
	3: {
		msg:        stanza.Message{ID: testIQID, Type: stanza.GroupChatMessage},
		canceled:   true,
		writesBody: true,
		err:        context.Canceled,
	},
}

func TestSendMessage(t *testing.T) {
	for i, tc := range sendMessageTests {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("SendMessageElement", func(t *testing.T) {
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

				resp, err := s.Client.SendMessageElement(ctx, tc.payload, tc.msg)
				if err != tc.err {
					t.Errorf("unexpected error, want=%q, got=%q", tc.err, err)
				}
				respMsg := stanza.Message{}
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
			t.Run("SendMessage", func(t *testing.T) {
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

				resp, err := s.Client.SendMessage(ctx, tc.msg.Wrap(tc.payload))
				if err != tc.err {
					t.Errorf("unexpected error, want=%q, got=%q", tc.err, err)
				}
				respMsg := stanza.Message{}
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
