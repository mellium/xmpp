// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type testReader []xml.Token

func (r *testReader) Token() (t xml.Token, err error) {
	tr := *r
	if len(tr) < 1 {
		return nil, io.EOF
	}
	t, *r = tr[0], tr[1:]
	return t, nil
}

var start = xml.StartElement{
	Name: xml.Name{Local: "ping"},
}

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
			iq := stanza.WrapIQ(&stanza.IQ{To: jid.MustParse(tc.to), Type: tc.typ}, tc.payload)
			if _, err := xmlstream.Copy(e, iq); err != tc.err {
				t.Errorf("Unexpected error: want=`%v', got=`%v'", tc.err, err)
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

type messageTest struct {
	to      string
	typ     stanza.MessageType
	payload xml.TokenReader
	out     string
	err     error
}

var messageTests = [...]messageTest{
	0: {
		to:      "new@example.net",
		payload: &testReader{},
	},
	1: {
		to:      "new@example.org",
		payload: &testReader{start, start.End()},
		out:     `<ping></ping>`,
		typ:     stanza.NormalMessage,
	},
}

func TestMessage(t *testing.T) {
	for i, tc := range messageTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b := new(bytes.Buffer)
			e := xml.NewEncoder(b)
			message := stanza.WrapMessage(jid.MustParse(tc.to), tc.typ, tc.payload)
			if _, err := xmlstream.Copy(e, message); err != tc.err {
				t.Errorf("Unexpected error: want=`%v', got=`%v'", tc.err, err)
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

type presenceTest struct {
	to      string
	typ     stanza.PresenceType
	payload xml.TokenReader
	out     string
	err     error
}

var presenceTests = [...]presenceTest{
	0: {
		to:      "new@example.net",
		payload: &testReader{},
	},
	1: {
		to:      "new@example.org",
		payload: &testReader{start, start.End()},
		out:     `<ping></ping>`,
		typ:     stanza.ProbePresence,
	},
}

func TestPresence(t *testing.T) {
	for i, tc := range presenceTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b := new(bytes.Buffer)
			e := xml.NewEncoder(b)
			presence := stanza.WrapPresence(jid.MustParse(tc.to), tc.typ, tc.payload)
			if _, err := xmlstream.Copy(e, presence); err != tc.err {
				t.Errorf("Unexpected error: want=`%v', got=`%v'", tc.err, err)
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
