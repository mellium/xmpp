// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

var exampleJID = jid.MustParse("example.net")

var wrapPresenceTests = [...]struct {
	to      jid.JID
	typ     stanza.PresenceType
	payload xml.TokenReader
	out     string
}{
	0: {out: "<presence></presence>"},
	1: {
		to:  exampleJID,
		out: `<presence to="example.net"></presence>`,
	},
	2: {
		typ: stanza.SubscribedPresence,
		out: `<presence type="subscribed"></presence>`,
	},
	3: {
		to:  exampleJID,
		typ: stanza.SubscribedPresence,
		out: `<presence to="example.net" type="subscribed"></presence>`,
	},
	4: {
		payload: &testReader{},
		out:     `<presence></presence>`,
	},
	5: {
		payload: &testReader{start, start.End()},
		out:     `<presence><ping></ping></presence>`,
	},
}

func TestWrapPresence(t *testing.T) {
	for i, tc := range wrapPresenceTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			e := xml.NewEncoder(buf)
			presence := stanza.WrapPresence(tc.to, tc.typ, tc.payload)
			_, err := xmlstream.Copy(e, presence)
			if err != nil {
				t.Fatalf("Error encoding stream: %q", err)
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("Error flushing stream: %q", err)
			}
			if s := buf.String(); s != tc.out {
				t.Fatalf("Wrong encoding:\nwant=\n%q,\ngot=\n%q", tc.out, s)
			}
		})
	}
}

func TestMarshalPresenceTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		presencetype stanza.PresenceType
		value        string
	}{
		0: {stanza.PresenceType(""), ""},
		1: {stanza.ErrorPresence, "error"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
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
		t.Run(strconv.Itoa(i), func(t *testing.T) {
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
