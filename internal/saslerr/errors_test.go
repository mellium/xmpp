// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package saslerr_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/saslerr"
	"mellium.im/xmpp/internal/xmpptest"
)

// Compile time tests that interfaces are satisfied
var (
	_ error               = saslerr.Failure{}
	_ error               = (*saslerr.Failure)(nil)
	_ xml.Marshaler       = saslerr.Failure{}
	_ xml.Marshaler       = (*saslerr.Failure)(nil)
	_ xml.Unmarshaler     = (*saslerr.Failure)(nil)
	_ xmlstream.Marshaler = (*saslerr.Failure)(nil)
	_ xmlstream.WriterTo  = (*saslerr.Failure)(nil)
	_ xml.Marshaler       = saslerr.Condition(0)
	_ xml.Marshaler       = (*saslerr.Condition)(nil)
	_ xml.Unmarshaler     = (*saslerr.Condition)(nil)
	_ xmlstream.Marshaler = (*saslerr.Condition)(nil)
	_ xmlstream.WriterTo  = (*saslerr.Condition)(nil)
)

func indirect(c saslerr.Condition) *saslerr.Condition {
	return &c
}

var encodingTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &saslerr.Failure{},
		XML:   `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"></failure>`,
	},
	1: {
		Value: &saslerr.Failure{
			Condition: saslerr.ConditionAborted,
		},
		XML: `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted></failure>`,
	},
	2: {
		Value: &saslerr.Failure{
			Condition: saslerr.ConditionAborted,
			Lang:      "pt-BR",
		},
		XML:         `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted></failure>`,
		NoUnmarshal: true,
	},
	3: {
		Value: &saslerr.Failure{
			Condition: saslerr.ConditionAborted,
			Lang:      "lir",
			Text:      "test",
		},
		XML: `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted><text xml:lang="lir">test</text></failure>`,
	},
	4: {
		Value: &saslerr.Failure{
			Condition: saslerr.ConditionAborted,
			Text:      "test",
		},
		XML: `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><aborted></aborted><text>test</text></failure>`,
	},
}

func TestEncoding(t *testing.T) {
	xmpptest.RunEncodingTests(t, encodingTestCases)
}

func TestConditionEncoding(t *testing.T) {
	var condEncodingTestCases = []xmpptest.EncodingTestCase{
		0: {
			Value:       indirect(saslerr.ConditionNone),
			NoUnmarshal: true,
		},
		1: {
			Value:       indirect(saslerr.Condition(100)),
			XML:         "",
			NoUnmarshal: true,
		},
		2: {
			Value:     indirect(saslerr.ConditionNone),
			XML:       "<badcondition/>",
			NoMarshal: true,
		},
	}
	for cond := saslerr.Condition(1); cond < saslerr.Condition(len(saslerr.ConditionIndex)-1); cond++ {
		cond := cond
		condEncodingTestCases = append(condEncodingTestCases, xmpptest.EncodingTestCase{
			Value: &cond,
			XML:   "<" + cond.String() + "></" + cond.String() + ">",
		})
	}
	xmpptest.RunEncodingTests(t, condEncodingTestCases)
}

func TestBadConditionString(t *testing.T) {
	s := saslerr.Condition(100).String()
	const expect = "Condition(100)"
	if s != expect {
		t.Errorf("got wrong condition string: want=%q, got=%q", expect, s)
	}
}

func TestErrorTextOrCondition(t *testing.T) {
	f := saslerr.Failure{
		Condition: saslerr.ConditionMechanismTooWeak,
		Text:      "Test",
		Lang:      "bn",
	}
	if f.Error() != f.Text {
		t.Error("Expected Error() to return the value of Text")
	}
	f = saslerr.Failure{
		Condition: saslerr.ConditionMechanismTooWeak,
	}
	if f.Error() != f.Condition.String() {
		t.Error("Expected Error() to return the value of Condition if no text")
	}
}
