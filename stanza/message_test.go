// Copyright 2015 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const testNS = "ns"

func TestMarshalMessageTypeAttr(t *testing.T) {
	for i, tc := range [...]struct {
		messagetype stanza.MessageType
		value       string
		err         error
	}{
		0: {stanza.MessageType(""), "normal", nil},
		1: {stanza.NormalMessage, "normal", nil},
		2: {stanza.ChatMessage, "chat", nil},
		3: {stanza.HeadlineMessage, "headline", nil},
		4: {stanza.ErrorMessage, "error", nil},
		5: {stanza.MessageType("foo"), "normal", nil},
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

			if err == nil && !bytes.Contains(b, []byte(fmt.Sprintf(`type="%s"`, tc.value))) {
				t.Errorf(`Expected output to contain type="%s", found: %s`, tc.value, b)
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

func TestMessageStartElement(t *testing.T) {
	to := jid.MustParse("to@example.net")
	from := jid.MustParse("from@example.net")
	msg := stanza.Message{
		XMLName: xml.Name{Space: "ns", Local: "badname"},
		ID:      "123",
		To:      to,
		From:    from,
		Lang:    "te",
		Type:    stanza.ChatMessage,
	}

	start := msg.StartElement()
	if start.Name.Local != "message" || start.Name.Space != testNS {
		t.Errorf("wrong value for name: want=%v, got=%v", xml.Name{Space: testNS, Local: "message"}, start.Name)
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

func TestMessageFromStartElement(t *testing.T) {
	for i, tc := range [...]xml.StartElement{
		0: {
			// Make sure that we're not validating the name.
			// This is deliberate in case we want to eg. use this to generate a
			// message in response to an IQ using the values from the IQ start
			// element.
			Name: xml.Name{Local: "iq", Space: testNS},
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
			msg, err := stanza.NewMessage(tc)
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
			exists, typeAttr := attr.Get(tc.Attr, "type")
			if exists == -1 {
				typeAttr = "normal"
			}
			if typeAttr != string(msg.Type) {
				t.Errorf("wrong value for type: want=%q, got=%q", typeAttr, msg.Type)
			}
		})
	}
}

func getLangAttr(start xml.StartElement) xml.Attr {
	var langAttr xml.Attr
	for _, attr := range start.Attr {
		if attr.Name.Local == "lang" && attr.Name.Space == ns.XML {
			langAttr = attr
			break
		}
	}
	return langAttr
}

func TestMessageError(t *testing.T) {
	msg := stanza.Message{
		To:   jid.MustParse("to"),
		From: jid.MustParse("from"),
	}.Error(stanza.Error{
		Condition: stanza.BadRequest,
	})
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, msg)
	if err != nil {
		t.Fatalf("error encoding stream: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing stream: %v", err)
	}
	const expected = `<message type="error" to="from" from="to"><error><bad-request xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"></bad-request></error></message>`
	if out := buf.String(); expected != out {
		t.Errorf("unexpected output:\nwant=%s,\n got=%s", expected, out)
	}
}
