// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var (
	_ error               = stanza.Error{}
	_ xmlstream.WriterTo  = stanza.Error{}
	_ xmlstream.Marshaler = stanza.Error{}

	simpleText = map[string]string{
		"": "test",
	}
)

var cmpTests = [...]struct {
	err    error
	target error
	is     bool
}{
	0: {
		err:    stanza.Error{},
		target: errors.New("test"),
	},
	1: {
		err:    stanza.Error{},
		target: stanza.Error{},
		is:     true,
	},
	2: {
		err:    stanza.Error{Type: stanza.Cancel},
		target: stanza.Error{},
		is:     true,
	},
	3: {
		err:    stanza.Error{Condition: stanza.UnexpectedRequest},
		target: stanza.Error{},
		is:     true,
	},
	4: {
		err:    stanza.Error{Type: stanza.Auth, Condition: stanza.UndefinedCondition},
		target: stanza.Error{},
		is:     true,
	},
	5: {
		err:    stanza.Error{Type: stanza.Cancel},
		target: stanza.Error{Type: stanza.Auth},
	},
	6: {
		err:    stanza.Error{Type: stanza.Auth},
		target: stanza.Error{Type: stanza.Auth},
		is:     true,
	},
	7: {
		err:    stanza.Error{Type: stanza.Continue, Condition: stanza.SubscriptionRequired},
		target: stanza.Error{Type: stanza.Continue},
		is:     true,
	},
	8: {
		err:    stanza.Error{Type: stanza.Continue},
		target: stanza.Error{Type: stanza.Continue, Condition: stanza.SubscriptionRequired},
	},
	9: {
		err:    stanza.Error{Condition: stanza.BadRequest},
		target: stanza.Error{Condition: stanza.Conflict},
	},
	10: {
		err:    stanza.Error{Condition: stanza.FeatureNotImplemented},
		target: stanza.Error{Condition: stanza.FeatureNotImplemented},
		is:     true,
	},
	11: {
		err:    stanza.Error{Type: stanza.Continue, Condition: stanza.Forbidden},
		target: stanza.Error{Condition: stanza.Forbidden},
		is:     true,
	},
	12: {
		err:    stanza.Error{Condition: stanza.Forbidden},
		target: stanza.Error{Type: stanza.Continue, Condition: stanza.Forbidden},
	},
}

func TestCmp(t *testing.T) {
	for i, tc := range cmpTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			is := errors.Is(tc.err, tc.target)
			if is != tc.is {
				t.Errorf("unexpected comparison, want=%t, got=%t", tc.is, is)
			}
		})
	}
}

func TestErrorReturnsCondition(t *testing.T) {
	s := stanza.Error{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	const expected = "Text"
	s = stanza.Error{Condition: "nope", Text: map[string]string{
		"": expected,
	}}
	if expected != s.Error() {
		t.Errorf("expected stanza error to return text %q but got %q", expected, s.Error())
	}
}

var errorEncodingTests = []xmpptest.EncodingTestCase{
	0: {
		Value:       &stanza.Error{},
		XML:         `<error><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
		NoUnmarshal: true,
	},
	1: {
		Value: &stanza.Error{Condition: stanza.UndefinedCondition},
		XML:   `<error><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
	},
	2: {
		Value: &stanza.Error{Condition: stanza.UnexpectedRequest},
		XML:   `<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
	},
	3: {
		Value: &stanza.Error{Type: stanza.Cancel, Condition: stanza.UnexpectedRequest},
		XML:   `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
	},
	4: {
		Value: &stanza.Error{Type: stanza.Wait, Condition: stanza.UndefinedCondition},
		XML:   `<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
	},
	5: {
		Value: &stanza.Error{Type: stanza.Modify, By: jid.MustParse("test@example.net"), Condition: stanza.SubscriptionRequired},
		XML:   `<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`,
	},
	6: {
		Value: &stanza.Error{Type: stanza.Continue, Condition: stanza.ServiceUnavailable, Text: simpleText},
		XML:   `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">test</text></error>`,
	},
	7: {XML: "", Value: &stanza.Error{}, Err: io.EOF, NoMarshal: true},
	8: {
		XML: `<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text></error>`,
		Value: &stanza.Error{Type: stanza.Auth, Condition: stanza.ResourceConstraint, Text: map[string]string{
			"en": "test",
		}},
	},
	9: {
		Value:     &stanza.Error{By: jid.JID{}, Condition: stanza.RemoteServerNotFound},
		XML:       `<error by=""><remote-server-not-found xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-not-found></error>`,
		NoMarshal: true,
	},
	10: {
		Value: &stanza.Error{Condition: stanza.Condition("other")},
		XML:   `<error><other xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></other></error>`,
	},
}

func TestEncodeError(t *testing.T) {
	xmpptest.RunEncodingTests(t, errorEncodingTests)
}

func TestWrapError(t *testing.T) {
	stanzaErr := stanza.Error{Condition: stanza.RecipientUnavailable, Text: map[string]string{
		"ac-u": "test",
	}}
	r := stanzaErr.Wrap(xmlstream.Wrap(nil, xml.StartElement{Name: xml.Name{Local: "foo"}}))
	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, r)
	if err != nil {
		t.Fatalf("error copying tokens: %v", err)
	}
	if err = e.Flush(); err != nil {
		t.Fatalf("error flushing buffer: %v", err)
	}
	const expected = `<error><recipient-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></recipient-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="ac-u">test</text><foo></foo></error>`
	if out := buf.String(); out != expected {
		t.Errorf("wrong output: want=%v, got=%v", expected, out)
	}
}
