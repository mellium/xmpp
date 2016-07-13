// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"fmt"
	"testing"

	"mellium.im/xmpp/jid"
)

var (
	_ error        = (*StanzaError)(nil)
	_ error        = StanzaError{}
	_ fmt.Stringer = (*errorType)(nil)
	_ fmt.Stringer = Auth
)

func TestErrorReturnsCondition(t *testing.T) {
	s := StanzaError{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	s = StanzaError{Condition: "nope", Text: "Text"}
	if s.Text != s.Error() {
		t.Errorf("Expected stanza error to return text `Text` but got %s", s.Error())
	}
}

func TestMarshalStanzaError(t *testing.T) {
	for _, data := range []struct {
		se  StanzaError
		xml string
		err bool
	}{
		{StanzaError{}, "", true},
		{StanzaError{Condition: UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		{StanzaError{Type: Cancel, Condition: UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		{StanzaError{Type: Wait, Condition: UndefinedCondition}, `<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`, false},
		{StanzaError{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, `<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`, false},
		{StanzaError{Type: Continue, Condition: ServiceUnavailable, Text: "test"}, `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="und">test</text></error>`, false},
	} {
		b, err := xml.Marshal(data.se)
		switch {
		case data.err && err == nil:
			t.Errorf("Expected an error when marshaling stanza error %v", data.se)
			continue
		case !data.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case string(b) != data.xml:
			t.Errorf("Expected marshaling stanza error %v to be `%s` but got `%s`.", data.se, data.xml, string(b))
		}
	}
}
