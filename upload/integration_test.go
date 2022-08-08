// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package upload_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/prosody"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/upload"
)

const uploadDomain = "upload.localhost"

func TestIntegrationUpload(t *testing.T) {
	prosodyRun := prosody.Test(context.TODO(), t,
		integration.Log(),
		prosody.Upload(uploadDomain),
		prosody.ListenC2S(),
	)
	prosodyRun(integrationUpload)
}

func integrationUpload(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
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
	var file = []byte("Old One was he	and his medicine was strong.")
	slot, err := upload.GetSlot(ctx, upload.File{
		Name: "incipit.txt",
		Size: len(file),
	}, jid.MustParse(uploadDomain), session)
	if err != nil {
		t.Fatalf("error getting slot: %v", err)
	}
	req, err := slot.Put(context.TODO(), bytes.NewReader(file))
	if err != nil {
		t.Fatalf("error creating put request: %v", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("error PUTing file: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("file upload failed with code %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	/* #nosec */
	defer resp.Body.Close()
	resp, err = client.Get(slot.GetURL.String())
	if err != nil {
		t.Fatalf("error fetching uploaded file: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("file download failed with code %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	/* #nosec */
	defer resp.Body.Close()
	fileOut, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error downloading file: %v", err)
	}
	if !bytes.Equal(fileOut, file) {
		t.Fatalf("file contents do not match: want=%q, got=%q", file, fileOut)
	}
}
