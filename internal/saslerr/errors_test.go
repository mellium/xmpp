// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package saslerr

import (
	"encoding/xml"
	"strconv"
	"testing"

	"golang.org/x/text/language"

	"mellium.im/xmlstream"
)

// Compile time tests that interfaces are satisfied
var (
	_ error               = Failure{}
	_ error               = (*Failure)(nil)
	_ xml.Marshaler       = Failure{}
	_ xml.Marshaler       = (*Failure)(nil)
	_ xml.Unmarshaler     = (*Failure)(nil)
	_ xmlstream.Marshaler = (*Failure)(nil)
	_ xmlstream.WriterTo  = (*Failure)(nil)
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
	if f.Error() != f.Condition.String() {
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
		{Failure{Condition: None}, `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"></failure>`, false},
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
	for i, test := range []struct {
		XML         string
		IntoFailure Failure
		Failure     Failure
		Err         bool
	}{
		0: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><temporary-auth-failure></temporary-auth-failure></failure>`,
			Failure{}, Failure{Condition: TemporaryAuthFailure}, false,
		},
		1: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><mechanism-too-weak></mechanism-too-weak><text xml:lang="pt-BR">Test</text></failure>`,
			Failure{}, Failure{Lang: language.BrazilianPortuguese, Text: "Test", Condition: MechanismTooWeak}, false,
		},
		2: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><malformed-request></malformed-request><text xml:lang="pt-BR">pt-BR</text><text xml:lang="en-US">en-US</text></failure>`,
			Failure{Lang: language.English}, Failure{Lang: language.AmericanEnglish, Text: "en-US", Condition: MalformedRequest}, false,
		},
		3: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><invalid-mechanism></invalid-mechanism><text xml:lang="NOPE">NO</text></failure>`,
			Failure{}, Failure{Condition: InvalidMechanism}, false,
		},
		4: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><invalid-authzid></invalid-authzid><text xml:lang="pt-BR">TEXT</text><text xml:lang="NOPE">NO</text></failure>`,
			Failure{Lang: language.English}, Failure{Lang: language.BrazilianPortuguese, Text: "TEXT", Condition: InvalidAuthzID}, false,
		},
		5: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><wat></wat></failure>`,
			Failure{}, Failure{}, false,
		},
		6: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><nope></wat></failure>`,
			Failure{}, Failure{}, true,
		},
		// The following test cases are really just for branch coverage in the big
		// switch; it should be simplified eventually so that they are not
		// necessary:
		7: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><incorrect-encoding></incorrect-encoding></failure>`,
			Failure{}, Failure{Condition: IncorrectEncoding}, false,
		},
		8: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><encryption-required></encryption-required></failure>`,
			Failure{}, Failure{Condition: EncryptionRequired}, false,
		},
		9: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><credentials-expired></credentials-expired></failure>`,
			Failure{}, Failure{Condition: CredentialsExpired}, false,
		},
		10: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><account-disabled></account-disabled></failure>`,
			Failure{}, Failure{Condition: AccountDisabled}, false,
		},
		11: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted></failure>`,
			Failure{}, Failure{Condition: Aborted}, false,
		},
		12: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><not-authorized></not-authorized></failure>`,
			Failure{}, Failure{Condition: NotAuthorized}, false,
		},
		13: {
			`<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"></failure>`,
			Failure{}, Failure{}, false,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := xml.Unmarshal([]byte(test.XML), &test.IntoFailure)
			switch {
			case test.Err && err == nil:
				t.Fatal("Expected unmarshal to error")
			case !test.Err && err != nil:
				t.Fatal(err)
			case err != nil:
				return
			}
			if test.IntoFailure != test.Failure {
				t.Errorf("Expected failure:\n%#v\nbut got:\n%#v", test.Failure, test.IntoFailure)
			}
		})
	}
}
