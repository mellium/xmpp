// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"encoding/xml"
	"testing"
)

// There is no room for variation on the starttls feature negotiation, so step
// through the list process token for token.
func TestStartTLSList(t *testing.T) {
	for _, req := range []bool{true, false} {
		stls := StartTLS(req)
		var b bytes.Buffer
		r, err := stls.List(context.Background(), &b)
		switch {
		case err != nil:
			t.Fatal(err)
		case r != req:
			t.Errorf("Expected StartTLS listing required to be %v but got %v", req, r)
		}

		d := xml.NewDecoder(&b)
		tok, err := d.Token()
		if err != nil {
			t.Fatal(err)
		}
		se := tok.(xml.StartElement)
		switch {
		case se.Name != xml.Name{Space: NSStartTLS, Local: "starttls"}:
			t.Errorf("Expected starttls to start with %+v token but got %+v", NSStartTLS, se.Name)
		case len(se.Attr) != 1:
			t.Errorf("Expected starttls start element to have 1 attribute (xmlns), but got %+v", se.Attr)
		}
		if req {
			tok, err = d.Token()
			if err != nil {
				t.Fatal(err)
			}
			se := tok.(xml.StartElement)
			switch {
			case se.Name != xml.Name{Space: NSStartTLS, Local: "required"}:
				t.Errorf("Expected required start element but got %+v", se)
			case len(se.Attr) > 0:
				t.Errorf("Expected starttls required to have no attributes but got %d", len(se.Attr))
			}
			tok, err = d.Token()
			ee := tok.(xml.EndElement)
			switch {
			case se.Name != xml.Name{Space: NSStartTLS, Local: "required"}:
				t.Errorf("Expected required end element but got %+v", ee)
			}
		}
		tok, err = d.Token()
		if err != nil {
			t.Fatal(err)
		}
		ee := tok.(xml.EndElement)
		switch {
		case se.Name != xml.Name{Space: NSStartTLS, Local: "starttls"}:
			t.Errorf("Expected starttls end element but got %+v", ee)
		}
	}
}

func TestStartTLSParse(t *testing.T) {
	stls := StartTLS(true)
	for _, test := range []struct {
		msg string
		req bool
		err bool
	}{
		{`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`, false, false},
		{`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"></starttls>`, false, false},
		{`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required/></starttls>`, true, false},
		{`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required></required></starttls>`, true, false},
		{`<endtls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`, false, true},
		{`<starttls xmlns="badurn"/>`, false, true},
	} {
		d := xml.NewDecoder(bytes.NewBufferString(test.msg))
		tok, _ := d.Token()
		se := tok.(xml.StartElement)
		req, _, err := stls.Parse(context.Background(), d, &se)
		switch {
		case test.err && (err == nil):
			t.Error("Expected starttls.Parse to error")
		case !test.err && (err != nil):
			t.Error(err)
		case req != test.req:
			t.Errorf("STARTTLS required was wrong; expected %v but got %v", test.req, req)
		}
	}
}
