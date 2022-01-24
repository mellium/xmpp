// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"mellium.im/xmpp/ibb"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

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
			ln := serverIBB.Listen(s.Server)
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
