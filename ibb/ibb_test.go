// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"mellium.im/xmpp/ibb"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var _ net.Listener = (*ibb.Listener)(nil)

func TestSendSelf(t *testing.T) {
	clientIBB := &ibb.Handler{}
	serverIBB := &ibb.Handler{}
	clientM := mux.New(
		stanza.NSClient,
		ibb.Handle(clientIBB),
	)
	serverM := mux.New(
		stanza.NSClient,
		ibb.Handle(serverIBB),
	)
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(clientM),
		xmpptest.ServerHandler(serverM),
	)

	const (
		iqPayload  = "There are two spiritual dangers in not owning a farm."
		msgPayload = "One is the danger of supposing that breakfast comes from the grocery, and the other that heat comes from the furnace."
	)
	recv := make(chan struct {
		Got string
		Err error
	})

	ln := serverIBB.Listen(s.Server)
	go func() {
		result := struct {
			Got string
			Err error
		}{}
		for {
			select {
			case <-recv:
				return
			default:
			}
			serverConn, err := ln.Accept()
			if err != nil {
				result.Err = fmt.Errorf("error listening for conn: %w", err)
				recv <- result
				return
			}
			b, err := io.ReadAll(serverConn)
			if err != nil {
				result.Err = fmt.Errorf("error reading full payload: %w", err)
				recv <- result
				return
			}
			result.Got = string(b)
			recv <- result
		}
	}()

	t.Run("iq", func(t *testing.T) {
		clientConn, err := clientIBB.Open(context.Background(), s.Client, s.Server.LocalAddr())
		if err != nil {
			t.Fatalf("error opening connection: %v", err)
		}
		_, err = io.WriteString(clientConn, iqPayload)
		if err != nil {
			t.Fatalf("error writing string: %v", err)
		}
		err = clientConn.Close()
		if err != nil {
			t.Fatalf("error closing conn: %v", err)
		}
		got := <-recv
		if got.Err != nil {
			t.Fatal(err)
		}
		if got.Got != iqPayload {
			t.Errorf("got wrong payload type: want=%q, got=%q", iqPayload, got.Got)
		}
	})
	t.Run("msg", func(t *testing.T) {
		clientConn, err := clientIBB.OpenIQ(context.Background(), stanza.IQ{To: s.Server.LocalAddr()}, s.Client, false, 5, "1234")
		if err != nil {
			t.Fatalf("error opening connection: %v", err)
		}
		_, err = io.WriteString(clientConn, msgPayload)
		if err != nil {
			t.Fatalf("error writing string: %v", err)
		}
		err = clientConn.Close()
		if err != nil {
			t.Fatalf("error closing conn: %v", err)
		}
		got := <-recv
		if got.Err != nil {
			t.Fatal(err)
		}
		if got.Got != msgPayload {
			t.Errorf("got wrong payload type: want=%q, got=%q", iqPayload, got.Got)
		}
	})
	close(recv)
}

func TestBufferFull(t *testing.T) {
	clientIBB := &ibb.Handler{}
	serverIBB := &ibb.Handler{}
	clientM := mux.New(
		stanza.NSClient,
		ibb.Handle(clientIBB),
	)
	serverM := mux.New(
		stanza.NSClient,
		ibb.Handle(serverIBB),
	)
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(clientM),
		xmpptest.ServerHandler(serverM),
	)

	accept := make(chan error)
	ln := serverIBB.Listen(s.Server)
	go func() {
		serverConn, err := ln.Accept()
		if err != nil {
			accept <- err
			return
		}
		serverConn.(*ibb.Conn).SetReadBuffer(10)
		close(accept)
	}()

	clientConn, err := clientIBB.OpenIQ(context.Background(), stanza.IQ{To: s.Server.LocalAddr()}, s.Client, true, 20, "1234")
	if err != nil {
		t.Fatalf("error opening connection: %v", err)
	}
	if err := <-accept; err != nil {
		t.Fatalf("error accepting connection: %v", err)
	}
	const payload = "One swallow does not make a summer, but one skein of geese, cleaving the murk of a March thaw, is the spring."
	// The write *may* return a resource-constraint (if it actually writes and
	// doesn't just buffer), flush and close however will definitely result in a
	// resource constraint since the buffer size is smaller than the data size and
	// we probably need to write less.
	_, err = io.WriteString(clientConn, payload)
	if err != nil && !errors.Is(err, stanza.Error{Type: stanza.Wait, Condition: stanza.ResourceConstraint}) {
		t.Fatalf("write expected resource-constraint error, got: %v", err)
	}
	err = clientConn.Flush()
	if !errors.Is(err, stanza.Error{Type: stanza.Wait, Condition: stanza.ResourceConstraint}) {
		t.Fatalf("flush expected resource-constraint error, got: %v", err)
	}
	err = clientConn.Close()
	if !errors.Is(err, stanza.Error{Type: stanza.Wait, Condition: stanza.ResourceConstraint}) {
		t.Fatalf("close expected resource-constraint error, got: %v", err)
	}
}

func TestExpect(t *testing.T) {
	clientIBB := &ibb.Handler{}
	serverIBB := &ibb.Handler{}
	clientM := mux.New(
		stanza.NSClient,
		ibb.Handle(clientIBB),
	)
	serverM := mux.New(
		stanza.NSClient,
		ibb.Handle(serverIBB),
	)
	s := xmpptest.NewClientServer(
		xmpptest.ClientHandler(clientM),
		xmpptest.ServerHandler(serverM),
	)

	accept := make(chan error)
	const sid = "1234"
	ln := serverIBB.Listen(s.Server)
	go func() {
		_, err := ln.Expect(context.Background(), jid.JID{}, sid)
		if err != nil {
			accept <- err
			return
		}
		close(accept)
	}()

	// This is a bit jank and only usable in tests.
	// See the function definition comments for details.
	ln.WaitExpect(context.Background(), jid.JID{}, sid)
	_, err := clientIBB.OpenIQ(context.Background(), stanza.IQ{To: s.Server.LocalAddr()}, s.Client, true, 20, sid)
	if err != nil {
		t.Fatalf("error opening connection: %v", err)
	}
	if err := <-accept; err != nil {
		t.Fatalf("error accepting connection: %v", err)
	}
}
