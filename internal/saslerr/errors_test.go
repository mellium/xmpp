// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package saslerr

import (
	"encoding/xml"
	"testing"

	"golang.org/x/text/language"
)

// Compile time tests that interfaces are satisfied
var (
	_ error           = Failure{}
	_ error           = (*Failure)(nil)
	_ xml.Marshaler   = Failure{}
	_ xml.Marshaler   = (*Failure)(nil)
	_ xml.Unmarshaler = (*Failure)(nil)
)

func TestErrorTextOrCondition(t *testing.T) {
	f := Failure{
		Condition: MechanismTooWeak,
		Text:      "Test",
		Lang:      language.CanadianFrench,
	}
	if f.Error() != f.Text {
		t.Error("Expected Error() to return the value of Text")
	}
	f = Failure{
		Condition: MechanismTooWeak,
	}
	if f.Error() != string(f.Condition) {
		t.Error("Expected Error() to return the value of Condition if no text")
	}
}

func TestMarshalCondition(t *testing.T) {
	for i, test := range []struct {
		Failure   Failure
		Marshaled string
		err       bool
	}{
		{
			Failure{
				Condition: MechanismTooWeak,
				Text:      "Test",
				Lang:      language.BrazilianPortuguese,
			},
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><mechanism-too-weak></mechanism-too-weak><text xml:lang="pt-BR">Test</text></failure>`,
			false,
		},
		{Failure{Condition: IncorrectEncoding}, `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><incorrect-encoding></incorrect-encoding></failure>`, false},
		{Failure{Condition: Aborted, Lang: language.Polish}, `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted></failure>`, false},
	} {
		b, err := xml.Marshal(test.Failure)
		switch {
		case test.err && err == nil:
			t.Errorf("Expected error when marshaling condition %d", i)
			continue
		case !test.err && err != nil:
			t.Error(err)
			continue
		case err != nil:
			continue
		case string(b) != test.Marshaled:
			t.Errorf("Expected %s but got %s", test.Marshaled, b)
		}
	}
}

func TestUnmarshalCondition(t *testing.T) {
	for _, test := range []struct {
		XML         string
		IntoFailure Failure
		Failure     Failure
		Err         bool
	}{
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><temporary-auth-failure></temporary-auth-failure></failure>`,
			Failure{}, Failure{Condition: TemporaryAuthFailure}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><mechanism-too-weak></mechanism-too-weak><text xml:lang="pt-BR">Test</text></failure>`,
			Failure{}, Failure{Lang: language.BrazilianPortuguese, Text: "Test", Condition: MechanismTooWeak}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><malformed-request></malformed-request><text xml:lang="pt-BR">pt-BR</text><text xml:lang="en-US">en-US</text></failure>`,
			Failure{Lang: language.English}, Failure{Lang: language.AmericanEnglish, Text: "en-US", Condition: MalformedRequest}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><invalid-mechanism></invalid-mechanism><text xml:lang="NOPE">NO</text></failure>`,
			Failure{}, Failure{Condition: InvalidMechanism}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><invalid-authzid></invalid-authzid><text xml:lang="pt-BR">TEXT</text><text xml:lang="NOPE">NO</text></failure>`,
			Failure{Lang: language.English}, Failure{Lang: language.BrazilianPortuguese, Text: "TEXT", Condition: InvalidAuthzID}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><wat></wat></failure>`,
			Failure{}, Failure{Condition: condition("wat")}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><nope></wat></failure>`,
			Failure{}, Failure{}, true,
		},
		// The following test cases are really just for branch coverage in the big
		// switch; it should be simplified eventually so that they are not
		// necessary:
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><incorrect-encoding></incorrect-encoding></failure>`,
			Failure{}, Failure{Condition: IncorrectEncoding}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><encryption-required></encryption-required></failure>`,
			Failure{}, Failure{Condition: EncryptionRequired}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><credentials-expired></credentials-expired></failure>`,
			Failure{}, Failure{Condition: CredentialsExpired}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><account-disabled></account-disabled></failure>`,
			Failure{}, Failure{Condition: AccountDisabled}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted></failure>`,
			Failure{}, Failure{Condition: Aborted}, false,
		},
		{
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><not-authorized></not-authorized></failure>`,
			Failure{}, Failure{Condition: NotAuthorized}, false,
		},
	} {
		err := xml.Unmarshal([]byte(test.XML), &test.IntoFailure)
		switch {
		case test.Err && err == nil:
			t.Fatal("Expected unmarshal to error")
		case !test.Err && err != nil:
			t.Fatal(err)
		case err != nil:
			continue
		}
		if test.IntoFailure != test.Failure {
			t.Errorf("Expected failure %#v but got %#v", test.Failure, test.IntoFailure)
		}
	}
}
