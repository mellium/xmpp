// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"fmt"
	"testing"

	"golang.org/x/text/language"
	"mellium.im/xmpp/jid"
)

var (
	_ error        = (*Error)(nil)
	_ error        = Error{}
	_ fmt.Stringer = (*errorType)(nil)
	_ fmt.Stringer = Auth
)

func TestErrorReturnsCondition(t *testing.T) {
	s := Error{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	s = Error{Condition: "nope", Text: "Text"}
	if s.Text != s.Error() {
		t.Errorf("Expected stanza error to return text `Text` but got %s", s.Error())
	}
}

func TestMarshalStanzaError(t *testing.T) {
	for _, data := range []struct {
		se  Error
		xml string
		err bool
	}{
		{Error{}, "", true},
		{Error{Condition: UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		{Error{Type: Cancel, Condition: UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		{Error{Type: Wait, Condition: UndefinedCondition}, `<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`, false},
		{Error{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, `<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`, false},
		{Error{Type: Continue, Condition: ServiceUnavailable, Text: "test"}, `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="und">test</text></error>`, false},
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

func TestUnmarshalStanzaError(t *testing.T) {
	for _, data := range []struct {
		xml  string
		lang language.Tag
		se   Error
		err  bool
	}{
		{"", language.Und, Error{}, true},
		{`<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
			language.Und, Error{Condition: UnexpectedRequest}, false},
		{`<error type="cancel"><registration-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></registration-required></error>`,
			language.Und, Error{Condition: RegistrationRequired}, false},
		{`<error type="cancel"><redirect xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></redirect></error>`,
			language.Und, Error{Type: Cancel, Condition: Redirect}, false},
		{`<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
			language.Und, Error{Type: Wait, Condition: UndefinedCondition}, false},
		{`<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`,
			language.Und, Error{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, false},
		{`<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="und">test</text></error>`,
			language.Und, Error{Type: Continue, Condition: ServiceUnavailable, Text: "test"}, false},
		{`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text></error>`,
			language.Und, Error{Type: Auth, Condition: ResourceConstraint, Text: "test", Lang: language.English}, false},
		{`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="de">German</text></error>`,
			language.German, Error{Type: Auth, Condition: ResourceConstraint, Text: "German", Lang: language.German}, false},
		{`<error type="auth"><remote-server-timeout xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-timeout><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="es">Spanish</text></error>`,
			language.LatinAmericanSpanish, Error{Type: Auth, Condition: RemoteServerTimeout, Text: "Spanish", Lang: language.Spanish}, false},
		{`<error by=""><remote-server-not-found xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-not-found></error>`,
			language.Und, Error{By: &jid.JID{}, Condition: RemoteServerNotFound}, false},
		{`<error><other xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></other></error>`,
			language.Und, Error{Condition: condition("other")}, false},
		{`<error><recipient-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></recipient-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="ac-u">test</text></error>`,
			language.Und, Error{Condition: RecipientUnavailable}, false},
	} {
		se2 := Error{Lang: data.lang}
		err := xml.Unmarshal([]byte(data.xml), &se2)
		j1, j2 := data.se.By, se2.By
		data.se.By = nil
		se2.By = nil
		switch {
		case data.err && err == nil:
			t.Errorf("Expected an error when unmarshaling stanza error `%s`", data.xml)
			continue
		case !data.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case !j1.Equal(j2):
			t.Errorf(`Expected by="%v" but got by="%v"`, j1, j2)
		case data.se.Lang != se2.Lang:
			// This case is included in the next one, but I wanted it to print
			// something nicer for languages…
			t.Errorf("Expected unmarshaled stanza error to have lang `%s` but got `%s`.", data.se.Lang, se2.Lang)
		case data.se != se2:
			t.Errorf("Expected unmarshaled stanza error `%#v` but got `%#v`.", data.se, se2)
		}
	}
}
