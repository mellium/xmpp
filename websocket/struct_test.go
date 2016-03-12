// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package websocket

import (
	"encoding/xml"
	"testing"
)

func TestUnmarshalOpen(t *testing.T) {
	open := open{}
	sopen := []byte(`<open xmlns="urn:ietf:params:xml:ns:xmpp-framing"
                         to="example.com"
                         version="1.0" />`)
	if err := xml.Unmarshal(sopen, &open); err != nil {
		t.Logf("Error unmarshaling open element: %s", err)
		t.Fail()
	}

	if open.To != "example.com" {
		t.Logf("Bad value for to: want=example.com, got=%s", open.To)
		t.Fail()
	}
	if open.Version != "1.0" {
		t.Logf("Bad value for to: want=1.0, got=%s", open.Version)
		t.Fail()
	}
}

func TestMarshalOpen(t *testing.T) {
	open := open{
		From:    "example.com",
		ID:      "++TR84Sm6A3hnt3Q065SnAbbk3Y=",
		Lang:    "en",
		Version: "1.0",
	}
	bopen, err := xml.Marshal(open)
	if err != nil {
		t.Logf("Failed to marshal open element: %v", err)
		t.Fail()
	}
	expected := `<open xmlns="urn:ietf:params:xml:ns:xmpp-framing" from="example.com" version="1.0" xml:lang="en" id="++TR84Sm6A3hnt3Q065SnAbbk3Y="></open>`
	if sopen := string(bopen); sopen != expected {
		t.Logf("Got wrong XML for open element: want=%v,\ngot=%v", expected, sopen)
		t.Fail()
	}
}

func TestUnmarshalClose(t *testing.T) {
	closews := close{}
	bclosews := []byte(`<close xmlns="urn:ietf:params:xml:ns:xmpp-framing"/>`)
	if err := xml.Unmarshal(bclosews, &closews); err != nil {
		t.Logf("Error unmarshaling close element: %s", err)
		t.Fail()
	}
}

func TestMarshalClose(t *testing.T) {
	closews := close{}
	bclosews, err := xml.Marshal(closews)
	if err != nil {
		t.Logf("Failed to marshal close element: %v", err)
		t.Fail()
	}
	expected := `<close xmlns="urn:ietf:params:xml:ns:xmpp-framing"></close>`
	if sclosews := string(bclosews); sclosews != expected {
		t.Logf("Got wrong XML for close element: want=%v,\ngot=%v", expected, sclosews)
		t.Fail()
	}
}
