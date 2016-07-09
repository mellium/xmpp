// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"
	"testing"

	"mellium.im/sasl"
)

func TestSASLPanicsNoMechanisms(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected call to SASL() with no mechanisms to panic")
		}
	}()
	_ = SASL()
}

func TestSASLList(t *testing.T) {
	var b bytes.Buffer
	s := SASL(sasl.Plain("", "user", "pass"), sasl.ScramSha256("user", "pass"))
	req, err := s.List(context.Background(), &b)
	switch {
	case err != nil:
		t.Fatal(err)
	case req != true:
		t.Error("Expected SASL to be a required feature")
	}

	// Mechanisms should be printed exactly thus:
	if !bytes.Contains((&b).Bytes(), []byte(`<mechanism>PLAIN</mechanism>`)) {
		t.Error("Expected mechanisms list to include PLAIN")
	}
	if !bytes.Contains((&b).Bytes(), []byte(`<mechanism>SCRAM-SHA-256</mechanism>`)) {
		t.Error("Expected mechanisms list to include SCRAM-SHA-256")
	}

	// The wrapper can be a bit more flexible as long as the mechanisms are there.
	d := xml.NewDecoder(&b)
	tok, err := d.Token()
	if err != nil {
		t.Fatal(err)
	}
	se := tok.(xml.StartElement)
	if se.Name.Local != "mechanisms" || se.Name.Space != NSSASL {
		t.Errorf("Unexpected name for mechanisms start element: %+v", se.Name)
	}
	// Skip two mechanisms
	tok, err = d.Token()
	if err != nil {
		t.Fatal(err)
	}
	d.Skip()
	tok, err = d.Token()
	if err != nil {
		t.Fatal(err)
	}
	d.Skip()

	// Check the end token.
	tok, err = d.Token()
	if err != nil {
		t.Fatal(err)
	}
	_ = tok.(xml.EndElement)
}
