// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

func TestBindList(t *testing.T) {
	buf := &bytes.Buffer{}
	bind := xmpp.BindResource()
	e := xml.NewEncoder(buf)
	start := xml.StartElement{Name: xml.Name{Space: ns.Bind, Local: "bind"}}
	req, err := bind.List(context.Background(), e, start)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Flush(); err != nil {
		t.Fatal(err)
	}
	if !req {
		t.Error("Bind must always be required")
	}
	if buf.String() != `<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"></bind>` {
		t.Errorf("Got unexpected value for bind listing: `%s`", buf.String())
	}
}

func TestBindParse(t *testing.T) {
	bind := xmpp.BindResource()
	for i, test := range []struct {
		XML string
		err bool
	}{
		0: {`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>`, false},
		1: {`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'></bind>`, false},
		2: {`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'>STUFF</bind>`, false},
		3: {`<bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><test/></bind>`, false},
		4: {`<notbind xmlns='urn:ietf:params:xml:ns:xmpp-bind'></notbind>`, true},
		5: {`<bind xmlns='notbindns'></bind>`, true},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {

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
				return
			case !test.err && err != nil:
				t.Error(err)
				return
			}
			if !req {
				t.Error("Expected parsed bind feature to be required")
			}
			if data != nil {
				t.Error("Expected bind data to be nil")
			}
		})
	}
}
func bindFunc(j jid.JID, s string) (jid.JID, error) {
	if s == "" {
		s = "empty"
	}
	return j.WithResource(s)
}

var bindTestCases = [...]featureTestCase{
	// BindCustom server tests
	0: {
		state:      xmpp.Received,
		sf:         xmpp.BindCustom(bindFunc),
		in:         `<iq type="set" id="123" to="example.net" from="test@example.net"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><resource>test</resource></bind></iq>`,
		out:        `<iq type="result" to="test@example.net" from="example.net" id="123"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>test@example.net/test</jid></bind></iq>`,
		finalState: xmpp.Ready,
	},
	1: {
		state:      xmpp.Received,
		sf:         xmpp.BindCustom(bindFunc),
		in:         `<iq type="set" id="1" to="example.net"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><resource>test</resource></bind></iq>`,
		out:        `<iq type="result" from="example.net" id="1"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>test@example.net/test</jid></bind></iq>`,
		finalState: xmpp.Ready,
	},
	2: {
		state:      xmpp.Received,
		sf:         xmpp.BindCustom(bindFunc),
		in:         `<iq type="set" id="2" from="me@example.net"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><resource>test</resource></bind></iq>`,
		out:        `<iq type="result" to="me@example.net" id="2"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>test@example.net/test</jid></bind></iq>`,
		finalState: xmpp.Ready,
	},
	3: {
		state:      xmpp.Received,
		sf:         xmpp.BindCustom(bindFunc),
		in:         `<iq type="set" id="123"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><resource></resource></bind></iq>`,
		out:        `<iq type="result" id="123"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>test@example.net/empty</jid></bind></iq>`,
		finalState: xmpp.Ready,
	},
	4: {
		state:      xmpp.Received,
		sf:         xmpp.BindCustom(bindFunc),
		in:         `<iq type="set" id="123"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"/></iq>`,
		out:        `<iq type="result" id="123"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>test@example.net/empty</jid></bind></iq>`,
		finalState: xmpp.Ready,
	},
}

func TestBind(t *testing.T) {
	runFeatureTests(t, bindTestCases[:])
}
