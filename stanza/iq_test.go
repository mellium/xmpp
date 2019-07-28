// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"bytes"
	"encoding"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var _ encoding.TextMarshaler = stanza.IQType("")

type iqTest struct {
	to      string
	typ     stanza.IQType
	payload xml.TokenReader
	out     string
	err     error
}

var iqTests = [...]iqTest{
	0: {
		to:      "new@example.net",
		payload: &testReader{},
	},
	1: {
		to:      "new@example.org",
		payload: &testReader{start, start.End()},
		out:     `<ping></ping>`,
		typ:     stanza.GetIQ,
	},
}

func TestIQ(t *testing.T) {
	for i, tc := range iqTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b := new(bytes.Buffer)
			e := xml.NewEncoder(b)
			iq := stanza.WrapIQ(stanza.IQ{To: jid.MustParse(tc.to), Type: tc.typ}, tc.payload)
			if _, err := xmlstream.Copy(e, iq); err != tc.err {
				t.Errorf("Unexpected error: want=`%v', got=`%v'", tc.err, err)
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("Error flushing: %q", err)
			}

			o := b.String()
			jidattr := fmt.Sprintf(`to="%s"`, tc.to)
			if !strings.Contains(o, jidattr) {
				t.Errorf("Expected output to have attr `%s',\ngot=`%s'", jidattr, o)
			}
			typeattr := fmt.Sprintf(`type="%s"`, string(tc.typ))
			if !strings.Contains(o, typeattr) {
				t.Errorf("Expected output to have attr `%s',\ngot=`%s'", typeattr, o)
			}
			if !strings.Contains(o, tc.out) {
				t.Errorf("Expected output to contain payload `%s',\ngot=`%s'", tc.out, o)
			}
		})
	}
}

func TestMarshalIQTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		iqtype stanza.IQType
		value  string
	}{
		0: {stanza.IQType(""), "get"},
		1: {stanza.GetIQ, "get"},
		2: {stanza.SetIQ, "set"},
		3: {stanza.ResultIQ, "result"},
		4: {stanza.ErrorIQ, "error"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b, err := xml.Marshal(stanza.IQ{Type: tc.iqtype})
			if err != nil {
				t.Fatal("Got unexpected error while marshaling IQ:", err)
			}

			if err == nil && !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.value))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.value, b)
			}
		})
	}
}

func TestUnmarshalIQTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		iq     string
		iqtype stanza.IQType
	}{
		0: {`<iq/>`, stanza.IQType("")},
		1: {`<iq type=""/>`, stanza.IQType("")},
		2: {`<iq type="get"/>`, stanza.GetIQ},
		3: {`<iq type="error"/>`, stanza.ErrorIQ},
		4: {`<iq type="result"/>`, stanza.ResultIQ},
		5: {`<iq type="set"/>`, stanza.SetIQ},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			iq := stanza.IQ{}
			switch err := xml.Unmarshal([]byte(tc.iq), &iq); {
			case err != nil:
				t.Errorf("Got unexpected error while unmarshaling IQ: %v", err)
			case tc.iqtype != iq.Type:
				t.Errorf("Wrong type when unmarshaling IQ: want=%s, got=%s", tc.iqtype, iq.Type)
			}
		})
	}
}
