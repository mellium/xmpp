// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"fmt"
	"testing"

	"golang.org/x/text/language"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

var (
	_ error               = (*Error)(nil)
	_ error               = Error{}
	_ xmlstream.WriterTo  = (*Error)(nil)
	_ xmlstream.WriterTo  = Error{}
	_ xmlstream.Marshaler = (*Error)(nil)
	_ xmlstream.Marshaler = Error{}
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
	for i, data := range [...]struct {
		se  Error
		xml string
		err bool
	}{
		0: {Error{}, "", true},
		1: {Error{Condition: UnexpectedRequest}, `<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		2: {Error{Type: Cancel, Condition: UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		3: {Error{Type: Wait, Condition: UndefinedCondition}, `<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`, false},
		4: {Error{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, `<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`, false},
		5: {Error{Type: Continue, Condition: ServiceUnavailable, Text: "test"}, `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="und">test</text></error>`, false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b, err := xml.Marshal(data.se)
			switch {
			case data.err && err == nil:
				t.Errorf("Expected an error when marshaling stanza error %v", data.se)
			case !data.err && err != nil:
				t.Error(err)
			case err != nil:
				return
			case string(b) != data.xml:
				t.Errorf("Expected marshaling stanza error '%v' to be:\n`%s`\nbut got:\n`%s`.", data.se, data.xml, string(b))
			}
		})
	}
}

func TestUnmarshalStanzaError(t *testing.T) {
	for i, data := range [...]struct {
		xml  string
		lang language.Tag
		se   Error
		err  bool
	}{
		0: {"", language.Und, Error{}, true},
		1: {`<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
			language.Und, Error{Condition: UnexpectedRequest}, false},
		2: {`<error type="cancel"><registration-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></registration-required></error>`,
			language.Und, Error{Type: Cancel, Condition: RegistrationRequired}, false},
		3: {`<error type="cancel"><redirect xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></redirect></error>`,
			language.Und, Error{Type: Cancel, Condition: Redirect}, false},
		4: {`<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
			language.Und, Error{Type: Wait, Condition: UndefinedCondition}, false},
		5: {`<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`,
			language.Und, Error{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, false},
		6: {`<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="und">test</text></error>`,
			language.Und, Error{Type: Continue, Condition: ServiceUnavailable, Text: "test"}, false},
		7: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text></error>`,
			language.Und, Error{Type: Auth, Condition: ResourceConstraint, Text: "test", Lang: language.English}, false},
		8: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="de">German</text></error>`,
			language.German, Error{Type: Auth, Condition: ResourceConstraint, Text: "German", Lang: language.German}, false},
		9: {`<error type="auth"><remote-server-timeout xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-timeout><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="es">Spanish</text></error>`,
			language.LatinAmericanSpanish, Error{Type: Auth, Condition: RemoteServerTimeout, Text: "Spanish", Lang: language.Spanish}, false},
		10: {`<error by=""><remote-server-not-found xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-not-found></error>`,
			language.Und, Error{By: &jid.JID{}, Condition: RemoteServerNotFound}, false},
		11: {`<error><other xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></other></error>`,
			language.Und, Error{Condition: Condition("other")}, false},
		12: {`<error><recipient-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></recipient-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="ac-u">test</text></error>`,
			language.Und, Error{Condition: RecipientUnavailable}, false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			se2 := Error{Lang: data.lang}
			err := xml.Unmarshal([]byte(data.xml), &se2)
			j1, j2 := data.se.By, se2.By
			data.se.By = nil
			se2.By = nil
			switch {
			case data.err && err == nil:
				t.Errorf("Expected an error when unmarshaling stanza error `%s`", data.xml)
			case !data.err && err != nil:
				t.Error(err)
			case err != nil:
				return
			case !j1.Equal(j2):
				t.Errorf(`Expected by="%v" but got by="%v"`, j1, j2)
			case data.se.Lang != se2.Lang:
				// This case is included in the next one, but I wanted it to print
				// something nicer for languages…
				t.Errorf("Expected unmarshaled stanza error to have lang `%s` but got `%s`.", data.se.Lang, se2.Lang)
			case data.se != se2:
				t.Errorf("Expected unmarshaled stanza error:\n`%#v`\nbut got:\n`%#v`", data.se, se2)
			}
		})
	}
}
