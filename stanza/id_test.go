// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza_test

import (
	"encoding/xml"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

const (
	testOrigin = `<origin-id xmlns="urn:xmpp:sid:0" id="abc"></origin-id>`
	testStanza = `<stanza-id xmlns="urn:xmpp:sid:0" id="abc" by="test@example.net"></stanza-id>`
)

var idTestCases = [...]struct {
	in     string
	origin string
	id     string
	ns     string
}{
	0: {
		in:     `<message xmlns="jabber:client"></message>`,
		origin: `<message xmlns="jabber:client">` + testOrigin + `</message>`,
		id:     `<message xmlns="jabber:client">` + testStanza + `</message>`,
		ns:     stanza.NSClient,
	},
	1: {
		in:     `<iq xmlns="jabber:client"></iq>`,
		origin: `<iq xmlns="jabber:client">` + testOrigin + `</iq>`,
		id:     `<iq xmlns="jabber:client">` + testStanza + `</iq>`,
		ns:     stanza.NSClient,
	},
	2: {
		in:     `<presence xmlns="jabber:client"></presence>`,
		origin: `<presence xmlns="jabber:client">` + testOrigin + `</presence>`,
		id:     `<presence xmlns="jabber:client">` + testStanza + `</presence>`,
		ns:     stanza.NSClient,
	},
	3: {
		in:     `<message xmlns="jabber:server"></message>`,
		origin: `<message xmlns="jabber:server">` + testOrigin + `</message>`,
		id:     `<message xmlns="jabber:server">` + testStanza + `</message>`,
		ns:     stanza.NSServer,
	},
	4: {
		in:     `<iq xmlns="jabber:server"></iq>`,
		origin: `<iq xmlns="jabber:server">` + testOrigin + `</iq>`,
		id:     `<iq xmlns="jabber:server">` + testStanza + `</iq>`,
		ns:     stanza.NSServer,
	},
	5: {
		in:     `<presence xmlns="jabber:server"></presence>`,
		origin: `<presence xmlns="jabber:server">` + testOrigin + `</presence>`,
		id:     `<presence xmlns="jabber:server">` + testStanza + `</presence>`,
		ns:     stanza.NSServer,
	},
	6: {
		in:     `<not-stanza><message xmlns="jabber:client"></message></not-stanza>`,
		origin: `<not-stanza><message xmlns="jabber:client"></message></not-stanza>`,
		id:     `<not-stanza><message xmlns="jabber:client"></message></not-stanza>`,
		ns:     stanza.NSClient,
	},
	7: {
		in:     `<not-stanza><iq xmlns="jabber:client"></iq></not-stanza>`,
		origin: `<not-stanza><iq xmlns="jabber:client"></iq></not-stanza>`,
		id:     `<not-stanza><iq xmlns="jabber:client"></iq></not-stanza>`,
		ns:     stanza.NSClient,
	},
	8: {
		in:     `<not-stanza><presence xmlns="jabber:client"></presence></not-stanza>`,
		origin: `<not-stanza><presence xmlns="jabber:client"></presence></not-stanza>`,
		id:     `<not-stanza><presence xmlns="jabber:client"></presence></not-stanza>`,
		ns:     stanza.NSClient,
	},
	9: {
		in:     `<presence xmlns="jabber:badns"></presence>`,
		origin: `<presence xmlns="jabber:badns"></presence>`,
		id:     `<presence xmlns="jabber:badns"></presence>`,
		ns:     stanza.NSClient,
	},
	10: {
		in:     `<presence xmlns="jabber:badns"></presence>`,
		origin: `<presence xmlns="jabber:badns">` + testOrigin + `</presence>`,
		id:     `<presence xmlns="jabber:badns">` + testStanza + `</presence>`,
		ns:     "jabber:badns",
	},
}

func TestAddID(t *testing.T) {
	idReplacer := regexp.MustCompile(`id="(.*?)"`)

	by := jid.MustParse("test@example.net")

	for i, tc := range idTestCases {
		addID := stanza.AddID(by, tc.ns)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("origin", func(t *testing.T) {
				r := stanza.AddOriginID(xml.NewDecoder(strings.NewReader(tc.in)), tc.ns)
				// Prevent duplicate xmlns attributes. See https://mellium.im/issue/75
				r = xmlstream.RemoveAttr(func(start xml.StartElement, attr xml.Attr) bool {
					return attr.Name.Local == "xmlns"
				})(r)
				var buf strings.Builder
				e := xml.NewEncoder(&buf)
				_, err := xmlstream.Copy(e, r)
				if err != nil {
					t.Fatalf("error copying xml stream: %v", err)
				}
				if err = e.Flush(); err != nil {
					t.Fatalf("error flushing stream: %v", err)
				}
				out := buf.String()
				// We need this to be testable, not random.
				out = idReplacer.ReplaceAllString(out, `id="abc"`)
				if out != tc.origin {
					t.Errorf("wrong output:\nwant=%v,\n got=%v", tc.origin, out)
				}
			})
			t.Run("stanza", func(t *testing.T) {
				r := addID(xml.NewDecoder(strings.NewReader(tc.in)))
				// Prevent duplicate xmlns attributes. See https://mellium.im/issue/75
				r = xmlstream.RemoveAttr(func(start xml.StartElement, attr xml.Attr) bool {
					return attr.Name.Local == "xmlns"
				})(r)
				var buf strings.Builder
				e := xml.NewEncoder(&buf)
				_, err := xmlstream.Copy(e, r)
				if err != nil {
					t.Fatalf("error copying xml stream: %v", err)
				}
				if err = e.Flush(); err != nil {
					t.Fatalf("error flushing stream: %v", err)
				}
				out := buf.String()
				// We need this to be testable, not random.
				out = idReplacer.ReplaceAllString(out, `id="abc"`)
				if out != tc.id {
					t.Errorf("wrong output:\nwant=%v,\n got=%v", tc.id, out)
				}
			})
		})
	}
}
