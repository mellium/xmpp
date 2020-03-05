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
	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/ns"
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
			iq := stanza.IQ{To: jid.MustParse(tc.to), Type: tc.typ}.Wrap(tc.payload)
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

func TestIQResult(t *testing.T) {
	iq := stanza.IQ{
		ID:   "123",
		To:   jid.MustParse("to@example.net"),
		From: jid.MustParse("from@example.net"),
		Type: stanza.SetIQ,
	}
	reply := iq.Result(xmlstream.Wrap(nil, xml.StartElement{Name: xml.Name{Local: "foo"}}))

	var b strings.Builder
	e := xml.NewEncoder(&b)
	_, err := xmlstream.Copy(e, reply)
	if err != nil {
		t.Fatalf("error copying tokens: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing encoder: %v", err)
	}

	const expected = `<iq type="result" to="from@example.net" from="to@example.net" id="123"><foo></foo></iq>`
	out := b.String()
	if out != expected {
		t.Errorf("want=%q, got=%q", expected, out)
	}
}

func TestIQStartElement(t *testing.T) {
	to := jid.MustParse("to@example.net")
	from := jid.MustParse("from@example.net")
	msg := stanza.IQ{
		XMLName: xml.Name{Space: "ns", Local: "badname"},
		ID:      "123",
		To:      to,
		From:    from,
		Lang:    "te",
		Type:    stanza.SetIQ,
	}

	start := msg.StartElement()
	if start.Name.Local != "iq" || start.Name.Space != testNS {
		t.Errorf("wrong value for name: want=%v, got=%v", xml.Name{Space: testNS, Local: "iq"}, start.Name)
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

func TestIQFromStartElement(t *testing.T) {
	langAttr := xml.Attr{Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: "lo"}

	// Make sure that we're not validating the name.
	const stanzaLocal = "message"
	start := xml.StartElement{
		Name: xml.Name{Local: stanzaLocal, Space: testNS},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: "123"},
			{Name: xml.Name{Local: "to"}, Value: "to@example.com"},
			{Name: xml.Name{Local: "from"}, Value: "from@example.com"},
			{Name: xml.Name{Local: "lang"}, Value: "de"},
			langAttr,
			{Name: xml.Name{Local: "type"}, Value: "chat"},
		},
	}
	msg, err := stanza.NewIQ(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.XMLName.Local != stanzaLocal {
		t.Errorf("wrong localname value: want=%q, got=%q", stanzaLocal, msg.XMLName.Local)
	}
	if msg.XMLName.Space != testNS {
		t.Errorf("wrong namespace value: want=%q, got=%q", testNS, msg.XMLName.Space)
	}
	if _, v := attr.Get(start.Attr, "id"); v != msg.ID {
		t.Errorf("wrong value for id: want=%q, got=%q", v, msg.ID)
	}
	if _, v := attr.Get(start.Attr, "to"); v != msg.To.String() {
		t.Errorf("wrong value for to: want=%q, got=%q", v, msg.To)
	}
	if _, v := attr.Get(start.Attr, "from"); v != msg.From.String() {
		t.Errorf("wrong value for from: want=%q, got=%q", v, msg.From)
	}
	if langAttr.Value != msg.Lang {
		t.Errorf("wrong value for xml:lang: want=%q, got=%q", langAttr.Value, msg.Lang)
	}
	if _, v := attr.Get(start.Attr, "type"); v != string(msg.Type) {
		t.Errorf("wrong value for type: want=%q, got=%q", v, msg.Type)
	}
}
