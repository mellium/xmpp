// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package component_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/component"
	"mellium.im/xmpp/jid"
)

const header = `<?xml version="1.0" encoding="UTF-8"?>`

// some is a sentinal error that represents that some error must occur.
type some struct{}

func (some) Error() string { return "Anything" }

type componentClientTest struct {
	server string
	client string
	err    error
}

var componentClientTests = [...]componentClientTest{
	0: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net'></stream:stream>`,
		err:    errors.New("component: expected acknowledgement or error start token from server"),
	},
	//0: {
	//	server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net'></stream:stream>`,
	//	err:    errors.New("component: expected server stream to contain stream ID"),
	//},
	1: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'>`,
		err:    &xml.SyntaxError{Line: 1, Msg: "unexpected EOF"},
	},
	2: {
		server: xml.Header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'>`,
		err:    errors.New("component: received unexpected token from server"),
	},
	3: {
		server: header + header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'>`,
		err:    errors.New("component: received unexpected proc inst from server"),
	},
	4: {
		server: `<stream xmlns='jabber:component:accept' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'>`,
		err:    errors.New("component: expected stream:stream from server"),
	},
	5: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'>test`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    errors.New("component: expected acknowledgement or error start token from server"),
	},
	6: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><error></error>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    errors.New(""),
	},
	7: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><wrong/>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    errors.New("component: unknown start element: {{jabber:component:accept wrong} []}"),
	},
	8: {
		server: header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
	},
	9: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
	},
	10: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><error><not-authorized xmlns="urn:ietf:params:xml:ns:xmpp-streams"/></error>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    errors.New("not-authorized"),
	},
	11: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id=''><error><not-authorized xmlns="urn:ietf:params:xml:ns:xmpp-streams"/></error>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>e5e9fa1ba31ecd1ae84f75caaa474f3a663f05f4</handshake>`,
		err:    errors.New("not-authorized"),
	},
	12: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id=''><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'>`,
		err:    errors.New("component: expected server stream to contain stream ID"),
	},
}

func TestComponent(t *testing.T) {
	addr := jid.MustParse("test@example.net")
	for i, tc := range componentClientTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			out := new(bytes.Buffer)
			in := strings.NewReader(tc.server)

			_, err := component.NewSession(ctx, addr, []byte("secret"), struct {
				io.Reader
				io.Writer
			}{
				Reader: in,
				Writer: out,
			})
			var errStr, tcErrStr string
			if err != nil {
				errStr = err.Error()
			}
			if tc.err != nil {
				tcErrStr = tc.err.Error()
			}
			if errStr != tcErrStr {
				t.Fatalf("unexpected error: want=%v, got=%v", tc.err, err)
			}

			if o := out.String(); len(o) < len(tc.client) || o[:len(tc.client)] != tc.client {
				t.Errorf("unexpected output:\nwant=%v,\n got=%v", tc.client, o)
			}
		})
	}
}
