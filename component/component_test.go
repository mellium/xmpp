// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package component_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	"mellium.im/xmpp/component"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
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
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net'>`,
		err:    some{}, // missing ID attr
	},
	1: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'>`,
		err:    &xml.SyntaxError{Line: 1, Msg: "unexpected EOF"},
	},
	2: {
		server: xml.Header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    some{}, // don't allow whitespace
	},
	3: {
		server: header + header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    some{}, // allow only a single XML header
	},
	4: {
		server: `<stream xmlns='jabber:component:accept' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    some{}, // must start with stream:stream
	},
	5: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'>test`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    some{}, // expect ack or error
	},
	6: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><error></error>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    stream.NotAuthorized, // expect not authorized if error reported
	},
	7: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><wrong/>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    some{}, // expect ack or error
	},
	8: {
		server: header + `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    nil,
	},
	9: {
		server: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' from='example.net' id='1234'><handshake></handshake>`,
		client: `<stream:stream xmlns='jabber:component:accept' xmlns:stream='http://etherx.jabber.org/streams' to='example.net'><handshake>32532c0f7dbf1253c095b18b18e36d38d94c1256</handshake>`,
		err:    nil,
	},
}

func TestComponent(t *testing.T) {
	addr := jid.MustParse("test@example.net")
	for i, tc := range componentClientTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var outLock sync.Mutex
			out := new(bytes.Buffer)

			cpr, spw := io.Pipe()
			spr, cpw := io.Pipe()

			rw := struct {
				io.Reader
				io.Writer
			}{
				Reader: cpr,
				Writer: cpw,
			}

			go func() {
				io.Copy(spw, strings.NewReader(tc.server))
				spw.Close()
			}()

			outLock.Lock()
			go func() {
				defer outLock.Unlock()
				io.Copy(out, spr)
			}()

			_, err := component.NewClientSession(ctx, addr, []byte("secret"), rw)
			if _, ok := tc.err.(some); (ok && err == nil) || (!ok && !reflect.DeepEqual(err, tc.err)) {
				t.Fatalf("Unexpected error, got='%v' want='%v'", err, tc.err)
			}
			cpw.Close()
			if err != nil {
				return
			}

			outLock.Lock()
			if o := out.String(); o[:len(tc.client)] != tc.client {
				t.Errorf("Unexpected output:\nGot:\n`%s`\nWant:\n`%s`\n", o, tc.client)
			}
		})
	}
}
