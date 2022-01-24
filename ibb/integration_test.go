// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package ibb_test

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/xml"
	"io"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/ibb"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/aioxmpp"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

//go:embed aioxmpp_integration_test.py
var ibbScript string

func TestIntegrationIBB(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		integration.LogXML(),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationAioxmpp)
}

func integrationAioxmpp(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	p := cmd.C2SPort()
	j, pass := cmd.User()

	t.Run("send", func(t *testing.T) {
		session, err := cmd.DialClient(ctx, j, t,
			xmpp.StartTLS(&tls.Config{
				InsecureSkipVerify: true,
			}),
			xmpp.SASL("", pass, sasl.Plain),
			xmpp.BindResource(),
		)
		if err != nil {
			t.Fatalf("error connecting: %v", err)
		}
		sid := attr.RandomID()
		aioxmppRun := aioxmpp.Test(context.TODO(), t,
			integration.Log(),
			aioxmpp.ConfigFile(aioxmpp.Config{
				JID:      j,
				Password: pass,
				Port:     p,
			}),
			aioxmpp.Import("RecvIBB", ibbScript),
			aioxmpp.Args("-j", session.LocalAddr().String()),
			aioxmpp.Args("-sid", sid),
		)
		aioxmppRun(integrationSend(session, sid))
	})

	t.Run("recv", func(t *testing.T) {
		session, err := cmd.DialClient(ctx, j, t,
			xmpp.StartTLS(&tls.Config{
				InsecureSkipVerify: true,
			}),
			xmpp.SASL("", pass, sasl.Plain),
			xmpp.BindResource(),
		)
		if err != nil {
			t.Fatalf("error connecting: %v", err)
		}
		aioxmppRun := aioxmpp.Test(context.TODO(), t,
			integration.Log(),
			aioxmpp.ConfigFile(aioxmpp.Config{
				JID:      j,
				Password: pass,
				Port:     p,
			}),
			aioxmpp.Import("SendIBB", ibbScript),
			aioxmpp.Args("-j", session.LocalAddr().String()),
		)
		aioxmppRun(integrationRecv(session))
	})
}

func integrationRecv(session *xmpp.Session) func(context.Context, *testing.T, *integration.Cmd) {
	return func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		ibbHandler := &ibb.Handler{}
		echo := make(chan string)
		go func() {
			m := mux.New(
				stanza.NSClient,
				ibb.Handle(ibbHandler),
				mux.MessageFunc("", xml.Name{Local: "doneibb"}, func(msg stanza.Message, r xmlstream.TokenReadEncoder) error {
					bodyMessage := struct {
						stanza.Message
						Body string `xml:"body"`
					}{}
					err := xml.NewTokenDecoder(r).Decode(&bodyMessage)
					if err != nil {
						return err
					}
					go func() {
						echo <- bodyMessage.Body
					}()
					return nil
				}),
			)
			err := session.Serve(m)
			if err != nil {
				t.Logf("error from serve: %v", err)
			}
		}()

		ln := ibbHandler.Listen(session)

		const (
			recvData = "Warren snores through the night like a bear—a bass to the treble of the loons."
			sendData = "It is the the stillness of a moose intending to appear."
		)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("error accepting IBB connection: %v", err)
		}
		_, err = io.WriteString(conn, sendData)
		if err != nil {
			t.Fatalf("error writing on received connection: %v", err)
		}
		recv, err := io.ReadAll(conn)
		if err != nil {
			t.Fatalf("error receiving data from IBB session: %v", err)
		}
		if string(recv) != recvData {
			t.Fatalf("read wrong data from other end of IBB proxy: want=%s, got=%s", recvData, recv)
		}
		sent := <-echo
		if sent != sendData {
			t.Fatalf("other end of IBB proxy read wrong data: want=%s, got=%s", sendData, sent)
		}
	}
}

func integrationSend(session *xmpp.Session, sid string) func(context.Context, *testing.T, *integration.Cmd) {
	return func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
		ibbHandler := &ibb.Handler{}
		j := make(chan jid.JID)
		echo := make(chan string)
		go func() {
			m := mux.New(
				stanza.NSClient,
				ibb.Handle(ibbHandler),
				mux.MessageFunc("", xml.Name{Local: "startibb"}, func(msg stanza.Message, _ xmlstream.TokenReadEncoder) error {
					j <- msg.From
					return nil
				}),
				mux.MessageFunc("", xml.Name{Local: "doneibb"}, func(msg stanza.Message, r xmlstream.TokenReadEncoder) error {
					bodyMessage := struct {
						stanza.Message
						Body string `xml:"body"`
					}{}
					err := xml.NewTokenDecoder(r).Decode(&bodyMessage)
					if err != nil {
						return err
					}
					go func() {
						echo <- bodyMessage.Body
					}()
					return nil
				}),
			)
			err := session.Serve(m)
			if err != nil {
				t.Logf("error from serve: %v", err)
			}
		}()

		to := <-j
		const (
			sendData = "Getting up too early is a vice habitual in horned owls, stars, geese, and freight trains."
			recvData = "I feel a deep security in the single-mindedness of freight trains."
		)
		conn, err := ibbHandler.OpenIQ(ctx, stanza.IQ{To: to}, session, true, 4096, sid)
		if err != nil {
			t.Fatalf("error establishing IBB session: %v", err)
		}
		_, err = io.WriteString(conn, sendData)
		if err != nil {
			t.Fatalf("error writing on received connection: %v", err)
		}
		err = conn.Close()
		if err != nil {
			t.Fatalf("error closing connection: %v", err)
		}
		recv, err := io.ReadAll(conn)
		if err != nil {
			t.Fatalf("error receiving data from IBB session: %v", err)
		}
		if string(recv) != recvData {
			t.Fatalf("read wrong data from other end of IBB proxy: want=%s, got=%s", recvData, recv)
		}
		sent := <-echo
		if sent != sendData {
			t.Fatalf("other end of IBB proxy read wrong data: want=%s, got=%s", sendData, sent)
		}
	}
}
