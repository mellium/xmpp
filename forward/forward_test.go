// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package forward_test

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/forward"
	"mellium.im/xmpp/stanza"
)

func TestWrap(t *testing.T) {
	r := forward.Wrap(stanza.Message{
		Type: stanza.NormalMessage,
	}, "foo", time.Time{},
		xmlstream.Wrap(nil, xml.StartElement{Name: xml.Name{Local: "foo"}}),
	)
	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	_, err := xmlstream.Copy(e, r)
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing: %v", err)
	}
	const expected = `<message type="normal"><body>foo</body><forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-01T00:00:00Z"></delay><foo></foo></forwarded></message>`
	if out := buf.String(); out != expected {
		t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

func TestMarshal(t *testing.T) {
	f := forward.Forwarded{}
	var buf strings.Builder
	e := xml.NewEncoder(&buf)
	_, err := f.WriteXML(e)
	if err != nil {
		t.Fatalf("error encoding: %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("error flushing: %v", err)
	}
	const expected = `<forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-01T00:00:00Z"></delay></forwarded>`
	if out := buf.String(); out != expected {
		t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, out)
	}
}

var unwrapValidTestCases = [...]struct {
	unwrappedXML string
	reason       string
	inXML        string
	noDelay      bool
}{
	0: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"></foo>`,
		reason:       "Test",
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><foo/></forwarded>`,
	},
	1: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"></foo>`,
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><foo/></forwarded>`,
		noDelay:      true,
	},
	2: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Inner</delay></foo>`,
		reason:       "Test",
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><foo><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Inner</delay></foo><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay></forwarded>`,
	},
	3: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Inner</delay></foo><delay xmlns="urn:xmpp:delay" xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay>`,
		reason:       "Test1",
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><foo><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Inner</delay></foo><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test1</delay><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay></forwarded>`,
	},
	4: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"></foo><delay xmlns="urn:xmpp:delay" xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay>`,
		reason:       "Test1",
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test1</delay><foo/><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay></forwarded>`,
	},
	5: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"></foo><delay xmlns="urn:xmpp:delay" xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay>`,
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test1</delay><foo/><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test2</delay></forwarded>`,
		noDelay:      true,
	},
	6: {
		unwrappedXML: `<foo xmlns="urn:xmpp:forward:0"></foo>`,
		inXML:        `<forwarded xmlns="urn:xmpp:forward:0"><foo/></forwarded>`,
	},
}

var unwrapInvalidTestCases = [...]struct {
	inXML string
}{
	0: {
		inXML: `<tag xmlns="urn:xmpp:forward:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><foo/></tag>`,
	},
	1: {
		inXML: `<forwarded xmlns="urn:xmpp:space:0"><delay xmlns="urn:xmpp:delay" stamp="0001-01-02T05:00:00Z">Test</delay><foo/></forwarded>`,
	},
}

func TestUnwrapDelay(t *testing.T) {
	for i, tc := range unwrapValidTestCases {
		t.Run(fmt.Sprintf("valid:%d", i), func(t *testing.T) {
			var del *delay.Delay
			if !tc.noDelay {
				del = &delay.Delay{}
			}
			r, err := forward.Unwrap(del, xml.NewDecoder(strings.NewReader(tc.inXML)))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var buf strings.Builder
			e := xml.NewEncoder(&buf)
			_, err = xmlstream.Copy(e, r)
			if err != nil {
				t.Fatalf("error encoding: %v", err)
			}
			err = e.Flush()
			if err != nil {
				t.Fatalf("error flushing: %v", err)
			}
			if out := buf.String(); out != tc.unwrappedXML {
				t.Errorf("wrong XML: want=%v, got=%v", tc.unwrappedXML, out)
			}
			if del != nil && del.Reason != tc.reason {
				t.Errorf("did not unmarshal delay: want=%v, got=%v", "Test", tc.reason)
			}
		})
	}

	for i, tc := range unwrapInvalidTestCases {
		t.Run(fmt.Sprintf("invalid:%d", i), func(t *testing.T) {
			_, err := forward.Unwrap(nil, xml.NewDecoder(strings.NewReader(tc.inXML)))
			if err == nil {
				t.Error("expected a non nil error")
			}
		})
	}
}
