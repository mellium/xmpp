// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmpp/internal/ns"
)

// There is no room for variation on the starttls feature negotiation, so step
// through the list process token for token.
func TestStartTLSList(t *testing.T) {
	for _, req := range []bool{true, false} {
		name := "optional"
		if req {
			name = "required"
		}
		t.Run(name, func(t *testing.T) {
			stls := StartTLS(req, nil)
			var b bytes.Buffer
			e := xml.NewEncoder(&b)
			start := xml.StartElement{Name: xml.Name{Space: ns.StartTLS, Local: "starttls"}}
			r, err := stls.List(context.Background(), e, start)
			switch {
			case err != nil:
				t.Fatal(err)
			case r != req:
				t.Errorf("Expected StartTLS listing required to be %v but got %v", req, r)
			}
			if err = e.Flush(); err != nil {
				t.Fatal(err)
			}

			d := xml.NewDecoder(&b)
			tok, err := d.Token()
			if err != nil {
				t.Fatal(err)
			}
			se := tok.(xml.StartElement)
			switch {
			case se.Name != xml.Name{Space: ns.StartTLS, Local: "starttls"}:
				t.Errorf("Expected starttls to start with %+v token but got %+v", ns.StartTLS, se.Name)
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
				case se.Name != xml.Name{Space: ns.StartTLS, Local: "required"}:
					t.Errorf("Expected required start element but got %+v", se)
				case len(se.Attr) > 0:
					t.Errorf("Expected starttls required to have no attributes but got %d", len(se.Attr))
				}
				tok, err = d.Token()
				ee := tok.(xml.EndElement)
				switch {
				case se.Name != xml.Name{Space: ns.StartTLS, Local: "required"}:
					t.Errorf("Expected required end element but got %+v", ee)
				}
			}
			tok, err = d.Token()
			if err != nil {
				t.Fatal(err)
			}
			ee := tok.(xml.EndElement)
			switch {
			case se.Name != xml.Name{Space: ns.StartTLS, Local: "starttls"}:
				t.Errorf("Expected starttls end element but got %+v", ee)
			}
		})
	}
}

func TestStartTLSParse(t *testing.T) {
	stls := StartTLS(true, nil)
	for i, test := range [...]struct {
		msg string
		req bool
		err bool
	}{
		0: {`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`, false, false},
		1: {`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"></starttls>`, false, false},
		2: {`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required/></starttls>`, true, false},
		3: {`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required></required></starttls>`, true, false},
		4: {`<endtls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`, false, true},
		5: {`<starttls xmlns="badurn"/>`, false, true},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
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
		})
	}
}

type nopRWC struct {
	io.Reader
	io.Writer
}

func (nopRWC) Close() error {
	return nil
}

func TestNegotiateServer(t *testing.T) {
	stls := StartTLS(true, &tls.Config{})
	var b bytes.Buffer
	c := &Session{state: Received, conn: newConn(nopRWC{&b, &b})}
	_, rw, err := stls.Negotiate(context.Background(), c, nil)
	switch {
	case err != nil:
		t.Fatal(err)
	case rw == nil:
		t.Fatal("Expected a new ReadWriter when negotiating STARTTLS as a server")
	}

	// The server should send a proceed element.
	proceed := struct {
		XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
	}{}
	d := xml.NewDecoder(&b)
	if err = d.Decode(&proceed); err != nil {
		t.Error(err)
	}
}

func TestNegotiateClient(t *testing.T) {
	for i, test := range [...]struct {
		responses []string
		err       bool
		rw        bool
		state     SessionState
	}{
		0: {[]string{`<proceed xmlns="badns"/>`}, true, false, Secure},
		1: {[]string{`<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, false, true, Secure},
		2: {[]string{`<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'></bad>`}, true, false, 0},
		3: {[]string{`<failure xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, false, false, 0},
		4: {[]string{`<failure xmlns='urn:ietf:params:xml:ns:xmpp-tls'></bad>`}, true, false, 0},
		5: {[]string{`</somethingbadhappened>`}, true, false, 0},
		6: {[]string{`<notproceedorfailure xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, true, false, 0},
		7: {[]string{`chardata not start element`}, true, false, 0},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			stls := StartTLS(true, &tls.Config{})
			r := strings.NewReader(strings.Join(test.responses, "\n"))
			var b bytes.Buffer
			c := &Session{conn: newConn(nopRWC{r, &b})}
			c.in.d = xml.NewDecoder(c.conn)
			mask, rw, err := stls.Negotiate(context.Background(), c, nil)
			switch {
			case test.err && err == nil:
				t.Error("Expected an error from starttls client negotiation")
				return
			case !test.err && err != nil:
				t.Error(err)
				return
			case test.err && err != nil:
				return
			case b.String() != `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`:
				t.Errorf("Expected client to send starttls element but got `%s`", b.String())
			case test.state != mask:
				t.Errorf("Expected session state mask %v but got %v", test.state, mask)
			case test.rw && rw == nil:
				t.Error("Expected a new ReadWriter when negotiating STARTTLS as a client")
			case !test.rw && rw != nil:
				t.Error("Did not expect a new ReadWriter when negotiating STARTTLS as a client")
			}
		})
	}
}
