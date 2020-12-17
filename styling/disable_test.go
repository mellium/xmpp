// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/styling"
)

var (
	_ xml.Marshaler         = styling.Unstyled{}
	_ xmlstream.Marshaler   = styling.Unstyled{}
	_ xmlstream.WriterTo    = styling.Unstyled{}
	_ xml.Unmarshaler       = (*styling.Unstyled)(nil)
	_ xmlstream.Transformer = styling.Disable
)

var disableTestCases = [...]struct {
	in  string
	out string
}{
	0: {},
	1: {
		in:  `<message xmlns="jabber:client"/>`,
		out: `<message xmlns="jabber:client"><unstyled xmlns="urn:xmpp:styling:0"></unstyled></message>`,
	},
	2: {
		in:  `<message xmlns="jabber:server"/><message xmlns="jabber:client"><body>test</body></message>`,
		out: `<message xmlns="jabber:server"><unstyled xmlns="urn:xmpp:styling:0"></unstyled></message><message xmlns="jabber:client"><body xmlns="jabber:client">test</body><unstyled xmlns="urn:xmpp:styling:0"></unstyled></message>`,
	},
	3: {
		in:  `<message xmlns="jabber:badns"/>`,
		out: `<message xmlns="jabber:badns"></message>`,
	},
}

func TestDisable(t *testing.T) {
	for i, tc := range disableTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := styling.Disable(xml.NewDecoder(strings.NewReader(tc.in)))
			// Prevent duplicate xmlns attributes. See https://mellium.im/issue/75
			r = xmlstream.RemoveAttr(func(start xml.StartElement, attr xml.Attr) bool {
				return (start.Name.Local == "message" || start.Name.Local == "iq") && attr.Name.Local == "xmlns"
			})(r)
			var buf strings.Builder
			e := xml.NewEncoder(&buf)
			_, err := xmlstream.Copy(e, r)
			if err != nil {
				t.Fatalf("error encoding: %v", err)
			}
			if err = e.Flush(); err != nil {
				t.Fatalf("error flushing: %v", err)
			}

			if out := buf.String(); tc.out != out {
				t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, out)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	err := e.Encode(struct {
		stanza.Message

		Unstyled styling.Unstyled
	}{})
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	if err = e.Flush(); err != nil {
		t.Fatalf("error flushing: %v", err)
	}

	const expected = `<message to="" from=""><unstyled xmlns="urn:xmpp:styling:0"></unstyled></message>`
	if out := buf.String(); expected != out {
		t.Errorf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

var unmarshalTestCases = [...]struct {
	in  string
	out bool
}{
	0: {
		in:  `<message><unstyled xmlns="urn:xmpp:styling:0"/></message>`,
		out: true,
	},
	1: {in: `<message><wrong xmlns="urn:xmpp:styling:0"/></message>`},
	2: {in: `<message><unstyled xmlns="urn:xmpp:wrongns"/></message>`},
}

func TestUnmarshal(t *testing.T) {
	for i, tc := range unmarshalTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := struct {
				stanza.Message
				Unstyled styling.Unstyled
			}{}
			err := xml.NewDecoder(strings.NewReader(tc.in)).Decode(&m)
			if err != nil {
				t.Errorf("error decoding: %v", err)
			}
			if m.Unstyled.Value != tc.out {
				t.Errorf("bad decode: want=%t, got=%t", tc.out, m.Unstyled.Value)
			}
		})
	}
}
