// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"testing"

	"mellium.im/xmpp/stanza"
)

func TestMarshalIQTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		iqtype stanza.IQType
		value  string
	}{
		0: {stanza.IQType(""), ""},
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

			if err == nil && !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.iqtype))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.iqtype, b)
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
