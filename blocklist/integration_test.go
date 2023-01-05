// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package blocklist_test

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
	"mellium.im/xmpp/blocklist"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

//go:embed mod_integration_report.lua
var modIntegrationReportLua []byte

func modIntegrationReport() integration.Option {
	const modName = "integration_report"
	return func(cmd *integration.Cmd) error {
		err := prosody.Modules(modName)(cmd)
		if err != nil {
			return err
		}
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := w.Write(modIntegrationReportLua)
			return err
		})(cmd)
	}
}

func TestIntegrationBlock(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
		prosody.Modules("blocklist"),
	)
	prosodyRun(integrationBlock)
}

func integrationBlock(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
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
	go func() {
		err := session.Serve(nil)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()

	// Fetch the block list and make sure it's empty.
	iter := blocklist.Fetch(ctx, session)
	if iter.Next() {
		t.Fatalf("blocklist already contains items")
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing initial iter: %v", err)
	}

	// Add an item to the block list, then fetch it again and make sure we get the
	// item back.
	var (
		a = jid.MustParse("a@example.net")
		b = jid.MustParse("b@example.net")
	)
	err = blocklist.Add(ctx, session, a, b)
	if err != nil {
		t.Fatalf("error adding JIDs to the block list: %v", err)
	}

	var jids []jid.JID
	iter = blocklist.Fetch(ctx, session)
	for iter.Next() {
		jids = append(jids, iter.JID())
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing iter: %v", err)
	}
	if len(jids) != 2 {
		t.Fatalf("got different number of JIDs than expected: want=%d, got=%d", 2, len(jids))
	}
}

func TestIntegrationReport(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.ListenC2S(),
		prosody.Modules("blocklist"),
		modIntegrationReport(),
	)
	prosodyRun(integrationReport)
}

func integrationReport(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	j, pass := cmd.User()
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
	go func() {
		err := session.Serve(nil)
		if err != nil {
			t.Logf("error from serve: %v", err)
		}
	}()

	// File a report and check that it gets blocked and that the server processed
	// the report.

	itemA := blocklist.Item{
		JID:       jid.MustParse("a@example.net"),
		Reason:    blocklist.ReasonAbuse,
		StanzaIDs: []stanza.ID{{ID: "test"}},
		Text:      "Test Report",
	}
	itemB := blocklist.Item{
		JID:    jid.MustParse("b@example.net"),
		Reason: blocklist.ReasonSpam,
	}
	err = blocklist.Report(ctx, session, itemA, itemB)
	if err != nil {
		t.Fatalf("error reporting JIDs: %v", err)
	}

	// Reporting JIDs should add them to the blocklist.
	var jids []jid.JID
	iter := blocklist.Fetch(ctx, session)
	for iter.Next() {
		jids = append(jids, iter.JID())
	}
	err = iter.Close()
	if err != nil {
		t.Fatalf("error closing iter: %v", err)
	}
	if len(jids) != 2 {
		t.Fatalf("got different number of JIDs than expected: want=%d, got=%d", 2, len(jids))
	}

	// The server should receive and process the report information.
	report := struct {
		XMLName xml.Name `xml:"report"`
		Item    []struct {
			XMLName xml.Name `xml:"item"`
			Text    string   `xml:"text,attr"`
			JID     string   `xml:"jid,attr"`
			Reason  string   `xml:"reason,attr"`
		} `xml:"item"`
	}{}
	err = session.UnmarshalIQElement(ctx, xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Space: "urn:mellium:integration", Local: "report"}},
	), stanza.IQ{
		Type: stanza.GetIQ,
	}, &report)
	if err != nil {
		t.Fatalf("error asking for previous report: %v", err)
	}
	if len(report.Item) != 2 {
		t.Fatalf("got different number of reports than expected: want=%d, got=%d", 2, len(jids))
	}
}
