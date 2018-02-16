// Copyright 2015 The Mellium Contributors.
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

func TestMarshalMessageTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		messagetype stanza.MessageType
		value       string
		err         error
	}{
		0: {stanza.MessageType(""), "", nil},
		1: {stanza.NormalMessage, "normal", nil},
		2: {stanza.ChatMessage, "chat", nil},
		3: {stanza.HeadlineMessage, "headline", nil},
		4: {stanza.ErrorMessage, "error", nil},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b, err := xml.Marshal(stanza.Message{Type: tc.messagetype})
			if err != tc.err {
				t.Fatalf("Got unexpected error while marshaling Message: want='%v', got='%v'", tc.err, err)
			}

			// Special case to check that empty values are omitted
			if string(tc.messagetype) == "" {
				if bytes.Contains(b, []byte("type")) {
					t.Fatalf(`Didn't expect output to contain type attribute, found: %s`, b)
				}
				return
			}

			if err == nil && !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.messagetype))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.messagetype, b)
			}
		})
	}
}

func TestUnmarshalMessageTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		message     string
		messagetype stanza.MessageType
	}{
		0: {`<message type="normal"/>`, stanza.NormalMessage},
		1: {`<message type="error"/>`, stanza.ErrorMessage},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			message := stanza.Message{}
			switch err := xml.Unmarshal([]byte(tc.message), &message); {
			case err != nil:
				t.Errorf("Got unexpected error while unmarshaling Message: %v", err)
			case tc.messagetype != message.Type:
				t.Errorf("Wrong type when unmarshaling Message: want=%s, got=%s", tc.messagetype, message.Type)
			}
		})
	}
}
