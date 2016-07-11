// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"
)

func TestBindList(t *testing.T) {
	buf := &bytes.Buffer{}
	bind := BindResource()
	req, err := bind.List(context.Background(), buf)
	if err != nil {
		t.Fatal(err)
	}
	if !req {
		t.Error("Bind must always be required")
	}
	if buf.String() != `<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>` {
		t.Errorf("Got unexpected value for bind listing: `%s`", buf.String())
	}
}

func TestBindParse(t *testing.T) {
	bind := BindResource()
	for _, test := range []struct {
		XML string
		err bool
	}{
		{`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>`, false},
		{`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'></bind>`, false},
		{`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'>STUFF</bind>`, false},
		{`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><test/></bind>`, false},
		{`<notbind xmlns='urn:ietf:params:xml:ns:xmpp-bind'></notbind>`, true},
		{`<bind xmlns='notbindns'></bind>`, true},
	} {
		// Run each test twice, once without a requested resource and once for a
		// requested resource (which should be ignored, making the results
		// identical).
		d := xml.NewDecoder(strings.NewReader(test.XML))
		tok, err := d.Token()
		if err != nil {
			// We screwed up the test string…
			panic(err)
		}
		start := tok.(xml.StartElement)
		req, data, err := bind.Parse(context.Background(), d, &start)
		switch {
		case test.err && err == nil:
			t.Error("Expected error from parse")
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		}
		if !req {
			t.Error("Expected parsed bind feature to be required")
		}
		if data != nil {
			t.Error("Expected bind data to be nil")
		}
	}
}
