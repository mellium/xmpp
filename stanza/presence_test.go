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
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/ns"
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
		out: `<presence type="subscribed" to="example.net"></presence>`,
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
			presence := stanza.Presence{To: tc.to, Type: tc.typ}.Wrap(tc.payload)
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

			if !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.value))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.value, b)
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

func TestPresenceStartElement(t *testing.T) {
	to := jid.MustParse("to@example.net")
	from := jid.MustParse("from@example.net")
	msg := stanza.Presence{
		XMLName: xml.Name{Space: "ns", Local: "badname"},
		ID:      "123",
		To:      to,
		From:    from,
		Lang:    "te",
		Type:    stanza.SubscribedPresence,
	}

	start := msg.StartElement()
	if start.Name.Local != "presence" || start.Name.Space != testNS {
		t.Errorf("wrong value for name: want=%v, got=%v", xml.Name{Space: testNS, Local: "presence"}, start.Name)
	}
	if _, v := attr.Get(start.Attr, "id"); v != msg.ID {
		t.Errorf("wrong value for id: want=%q, got=%q", msg.ID, v)
	}
	if _, v := attr.Get(start.Attr, "to"); v != msg.To.String() {
		t.Errorf("wrong value for to: want=%q, got=%q", msg.To, v)
	}
	if _, v := attr.Get(start.Attr, "from"); v != msg.From.String() {
		t.Errorf("wrong value for from: want=%q, got=%q", msg.From, v)
	}
	if i, v := attr.Get(start.Attr, "lang"); v != msg.Lang || start.Attr[i].Name.Space != ns.XML {
		t.Errorf("wrong value for xml:lang: want=%q, got=%q", xml.Attr{
			Name:  xml.Name{Space: ns.XML, Local: "lang"},
			Value: msg.Lang,
		}, xml.Attr{
			Name:  start.Attr[i].Name,
			Value: v,
		})
	}
	if _, v := attr.Get(start.Attr, "type"); v != string(msg.Type) {
		t.Errorf("wrong value for type: want=%q, got=%q", msg.Type, v)
	}
}

func TestPresenceFromStartElement(t *testing.T) {
	for i, tc := range [...]xml.StartElement{
		0: {
			// Make sure that we're not validating the name.
			Name: xml.Name{Local: "message", Space: testNS},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "id"}, Value: "123"},
				{Name: xml.Name{Local: "to"}, Value: "to@example.com"},
				{Name: xml.Name{Local: "from"}, Value: "from@example.com"},
				{Name: xml.Name{Local: "lang"}, Value: "de"},
				{Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: "lo"},
				{Name: xml.Name{Local: "type"}, Value: "chat"},
			},
		},
		1: {
			Name: xml.Name{Local: "msg", Space: testNS},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "to"}, Value: ""},
				{Name: xml.Name{Local: "from"}, Value: ""},
			},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			msg, err := stanza.NewPresence(tc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if msg.XMLName.Local != tc.Name.Local {
				t.Errorf("wrong localname value: want=%q, got=%q", tc.Name.Local, msg.XMLName.Local)
			}
			if msg.XMLName.Space != testNS {
				t.Errorf("wrong namespace value: want=%q, got=%q", testNS, msg.XMLName.Space)
			}
			if _, v := attr.Get(tc.Attr, "id"); v != msg.ID {
				t.Errorf("wrong value for id: want=%q, got=%q", v, msg.ID)
			}
			if _, v := attr.Get(tc.Attr, "to"); v != msg.To.String() {
				t.Errorf("wrong value for to: want=%q, got=%q", v, msg.To)
			}
			if _, v := attr.Get(tc.Attr, "from"); v != msg.From.String() {
				t.Errorf("wrong value for from: want=%q, got=%q", v, msg.From)
			}
			langAttr := getLangAttr(tc)
			if langAttr.Value != msg.Lang {
				t.Errorf("wrong value for xml:lang: want=%q, got=%q", langAttr.Value, msg.Lang)
			}
			if _, v := attr.Get(tc.Attr, "type"); v != string(msg.Type) {
				t.Errorf("wrong value for type: want=%q, got=%q", v, msg.Type)
			}
		})
	}
}

func TestPresenceError(t *testing.T) {
	pres := stanza.Presence{
		To:   jid.MustParse("to"),
		From: jid.MustParse("from"),
	}.Error(stanza.Error{
		Condition: stanza.BadRequest,
	})
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, pres)
	if err != nil {
		t.Fatalf("error encoding stream: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing stream: %v", err)
	}
	const expected = `<presence type="error" to="from" from="to"><error><bad-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></bad-request></error></presence>`
	if out := buf.String(); expected != out {
		t.Errorf("unexpected output:\nwant=%s,\n got=%s", expected, out)
	}
}
