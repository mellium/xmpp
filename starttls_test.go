// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"mellium.im/xmpp/ns"
)

// There is no room for variation on the starttls feature negotiation, so step
// through the list process token for token.
func TestStartTLSList(t *testing.T) {
	for _, req := range []bool{true, false} {
		stls := StartTLS(req)
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

type nopRWC struct {
	io.Reader
	io.Writer
}

func (nopRWC) Close() error {
	return nil
}

var _ net.Conn = dummyConn{}

type dummyConn struct {
	io.ReadWriteCloser
}

func (dummyConn) LocalAddr() net.Addr {
	return nil
}

func (dummyConn) RemoteAddr() net.Addr {
	return nil
}

func (dummyConn) SetDeadline(t time.Time) error {
	return nil
}

func (dummyConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (dummyConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// We can't create a tls.Client or tls.Server for a generic RWC, so ensure that
// we fail (with a specific error) if this is the case.
func TestNegotiationFailsForNonNetSession(t *testing.T) {
	stls := StartTLS(true)
	var b bytes.Buffer
	_, _, err := stls.Negotiate(context.Background(), &Session{rwc: nopRWC{&b, &b}}, nil)
	if err != ErrTLSUpgradeFailed {
		t.Errorf("Expected error `%v` but got `%v`", ErrTLSUpgradeFailed, err)
	}
}

func TestNegotiateServer(t *testing.T) {
	stls := StartTLS(true)
	var b bytes.Buffer
	c := &Session{state: Received, conn: dummyConn{nopRWC{&b, &b}}, config: &Config{TLSConfig: &tls.Config{}}}
	c.rwc = c.conn
	_, rwc, err := stls.Negotiate(context.Background(), c, nil)
	switch {
	case err != nil:
		t.Fatal(err)
	case rwc == nil:
		t.Fatal("Expected a new RWC when negotiating STARTTLS as a server")
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
	for _, test := range []struct {
		responses []string
		err       bool
		rwc       bool
		state     SessionState
	}{
		{[]string{`<proceed xmlns="badns"/>`}, true, false, Secure},
		{[]string{`<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, false, true, Secure},
		{[]string{`<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'></bad>`}, true, false, 0},
		{[]string{`<failure xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, false, false, 0},
		{[]string{`<failure xmlns='urn:ietf:params:xml:ns:xmpp-tls'></bad>`}, true, false, 0},
		{[]string{`</somethingbadhappened>`}, true, false, 0},
		{[]string{`<notproceedorfailure xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`}, true, false, 0},
		{[]string{`chardata not start element`}, true, false, 0},
	} {
		stls := StartTLS(true)
		r := strings.NewReader(strings.Join(test.responses, "\n"))
		var b bytes.Buffer
		c := &Session{conn: dummyConn{nopRWC{r, &b}}, config: &Config{TLSConfig: &tls.Config{}}}
		c.rwc = c.conn
		c.in.d = xml.NewDecoder(c.rwc)
		mask, rwc, err := stls.Negotiate(context.Background(), c, nil)
		switch {
		case test.err && err == nil:
			t.Error("Expected an error from starttls client negotiation")
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case test.err && err != nil:
			continue
		case b.String() != `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`:
			t.Errorf("Expected client to send starttls element but got `%s`", b.String())
		case test.state != mask:
			t.Errorf("Expected session state mask %v but got %v", test.state, mask)
		case test.rwc && rwc == nil:
			t.Error("Expected a new RWC when negotiating STARTTLS as a client")
		case !test.rwc && rwc != nil:
			t.Error("Did not expect a new RWC when negotiating STARTTLS as a client")
		}
	}
}
