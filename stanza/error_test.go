// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var (
	_ error               = (*stanza.Error)(nil)
	_ error               = stanza.Error{}
	_ xmlstream.WriterTo  = (*stanza.Error)(nil)
	_ xmlstream.WriterTo  = stanza.Error{}
	_ xmlstream.Marshaler = (*stanza.Error)(nil)
	_ xmlstream.Marshaler = stanza.Error{}

	simpleText = map[string]string{
		"": "test",
	}
)

func TestErrorReturnsCondition(t *testing.T) {
	s := stanza.Error{Condition: "leprosy"}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return condition `leprosy` but got %s", s.Error())
	}
	s = stanza.Error{Condition: "nope", Text: map[string]string{
		"": "Text",
	}}
	if string(s.Condition) != s.Error() {
		t.Errorf("Expected stanza error to return text `Text` but got %s", s.Error())
	}
}

func TestMarshalStanzaError(t *testing.T) {
	for i, data := range [...]struct {
		se  stanza.Error
		xml string
		err bool
	}{
		0: {stanza.Error{}, "", true},
		1: {stanza.Error{Condition: stanza.UnexpectedRequest}, `<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		2: {stanza.Error{Type: stanza.Cancel, Condition: stanza.UnexpectedRequest}, `<error type="cancel"><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`, false},
		3: {stanza.Error{Type: stanza.Wait, Condition: stanza.UndefinedCondition}, `<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`, false},
		4: {stanza.Error{Type: stanza.Modify, By: jid.MustParse("test@example.net"), Condition: stanza.SubscriptionRequired}, `<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`, false},
		5: {stanza.Error{Type: stanza.Continue, Condition: stanza.ServiceUnavailable, Text: simpleText}, `<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">test</text></error>`, false},
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
		se  stanza.Error
		err bool
	}{
		0: {"", stanza.Error{}, true},
		1: {`<error><unexpected-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></unexpected-request></error>`,
			stanza.Error{Condition: stanza.UnexpectedRequest}, false},
		2: {`<error type="cancel"><registration-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></registration-required></error>`,
			stanza.Error{Type: stanza.Cancel, Condition: stanza.RegistrationRequired}, false},
		3: {`<error type="cancel"><redirect xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></redirect></error>`,
			stanza.Error{Type: stanza.Cancel, Condition: stanza.Redirect}, false},
		4: {`<error type="wait"><undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></undefined-condition></error>`,
			stanza.Error{Type: stanza.Wait, Condition: stanza.UndefinedCondition}, false},
		5: {`<error type="modify" by="test@example.net"><subscription-required xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></subscription-required></error>`,
			stanza.Error{Type: stanza.Modify, By: jid.MustParse("test@example.net"), Condition: stanza.SubscriptionRequired}, false},
		6: {`<error type="continue"><service-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></service-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas">test</text></error>`,
			stanza.Error{Type: stanza.Continue, Condition: stanza.ServiceUnavailable, Text: simpleText}, false},
		7: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text></error>`,
			stanza.Error{Type: stanza.Auth, Condition: stanza.ResourceConstraint, Text: map[string]string{
				"en": "test",
			}}, false},
		8: {`<error type="auth"><resource-constraint xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></resource-constraint><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="de">German</text></error>`,
			stanza.Error{Type: stanza.Auth, Condition: stanza.ResourceConstraint, Text: map[string]string{
				"en": "test",
				"de": "German",
			}}, false},
		9: {`<error type="auth"><remote-server-timeout xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-timeout><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="en">test</text><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="es">Spanish</text></error>`,
			stanza.Error{Type: stanza.Auth, Condition: stanza.RemoteServerTimeout, Text: map[string]string{
				"en": "test",
				"es": "Spanish",
			}}, false},
		10: {`<error by=""><remote-server-not-found xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></remote-server-not-found></error>`,
			stanza.Error{By: jid.JID{}, Condition: stanza.RemoteServerNotFound}, false},
		11: {`<error><other xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></other></error>`,
			stanza.Error{Condition: stanza.Condition("other")}, false},
		12: {`<error><recipient-unavailable xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></recipient-unavailable><text xmlns="urn:ietf:params:xml:ns:xmpp-stanzas" xml:lang="ac-u">test</text></error>`,
			stanza.Error{Condition: stanza.RecipientUnavailable, Text: map[string]string{
				"ac-u": "test",
			}}, false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			se2 := stanza.Error{}
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
