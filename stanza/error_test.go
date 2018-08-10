// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

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

	simpleText = map[string]string{
		"": "test",
	}
)

func TestErrorReturnsCondition(t *testing.T) {
	s := Error{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	s = Error{Condition: "nope", Text: map[string]string{
		"": "Text",
	}}
	if string(s.Condition) != s.Error() {
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
		5: {Error{Type: Continue, Condition: ServiceUnavailable, Text: simpleText}, `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">test</text></error>`, false},
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
		xml string
		se  Error
		err bool
	}{
		0: {"", Error{}, true},
		1: {`<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
			Error{Condition: UnexpectedRequest}, false},
		2: {`<error type="cancel"><registration-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></registration-required></error>`,
			Error{Type: Cancel, Condition: RegistrationRequired}, false},
		3: {`<error type="cancel"><redirect xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></redirect></error>`,
			Error{Type: Cancel, Condition: Redirect}, false},
		4: {`<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
			Error{Type: Wait, Condition: UndefinedCondition}, false},
		5: {`<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`,
			Error{Type: Modify, By: jid.MustParse("test@example.net"), Condition: SubscriptionRequired}, false},
		6: {`<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">test</text></error>`,
			Error{Type: Continue, Condition: ServiceUnavailable, Text: simpleText}, false},
		7: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text></error>`,
			Error{Type: Auth, Condition: ResourceConstraint, Text: map[string]string{
				"en": "test",
			}}, false},
		8: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="de">German</text></error>`,
			Error{Type: Auth, Condition: ResourceConstraint, Text: map[string]string{
				"en": "test",
				"de": "German",
			}}, false},
		9: {`<error type="auth"><remote-server-timeout xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-timeout><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="es">Spanish</text></error>`,
			Error{Type: Auth, Condition: RemoteServerTimeout, Text: map[string]string{
				"en": "test",
				"es": "Spanish",
			}}, false},
		10: {`<error by=""><remote-server-not-found xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-not-found></error>`,
			Error{By: jid.JID{}, Condition: RemoteServerNotFound}, false},
		11: {`<error><other xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></other></error>`,
			Error{Condition: Condition("other")}, false},
		12: {`<error><recipient-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></recipient-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="ac-u">test</text></error>`,
			Error{Condition: RecipientUnavailable, Text: map[string]string{
				"ac-u": "test",
			}}, false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			se2 := Error{}
			err := xml.Unmarshal([]byte(data.xml), &se2)
			j1, j2 := data.se.By, se2.By
			data.se.By = jid.JID{}
			se2.By = jid.JID{}
			switch {
			case data.err && err == nil:
				t.Errorf("Expected an error when unmarshaling stanza error `%s`", data.xml)
			case !data.err && err != nil:
				t.Error(err)
			case err != nil:
				return
			case !j1.Equal(j2):
				t.Errorf(`Expected by="%v" but got by="%v"`, j1, j2)
			case !reflect.DeepEqual(data.se, se2):
				t.Errorf("Expected unmarshaled stanza error:\n`%#v`\nbut got:\n`%#v`", data.se, se2)
			}
		})
	}
}
