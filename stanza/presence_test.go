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

func TestMarshalPresenceTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		presencetype stanza.PresenceType
		value        string
	}{
		0: {stanza.PresenceType(""), ""},
		1: {stanza.ErrorPresence, "error"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b, err := xml.Marshal(stanza.Presence{Type: tc.presencetype})
			if err != nil {
				t.Fatal("Unexpected error while marshaling:", err)
			}

			// Special case empty presence to make sure its omitted.
			if string(tc.presencetype) == "" {
				if bytes.Contains(b, []byte("type=")) {
					t.Fatalf(`Expected empty presence type to be omitted, found: %s`, b)
				}
				return
			}

			if !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.presencetype))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.presencetype, b)
			}
		})
	}
}

func TestUnmarshalPresenceTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		presence     string
		presencetype stanza.PresenceType
	}{
		0: {`<presence/>`, stanza.PresenceType("")},
		1: {`<presence type=""/>`, stanza.PresenceType("")},
		2: {`<presence type="probe"/>`, stanza.ProbePresence},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			presence := stanza.Presence{}
			switch err := xml.Unmarshal([]byte(tc.presence), &presence); {
			case err != nil:
				t.Error("Got unexpected error while unmarshaling Presence:", err)
			case tc.presencetype != presence.Type:
				t.Errorf("Wrong type when unmarshaling Presence: want=%s, got=%s", tc.presencetype, presence.Type)
			}
		})
	}
}
