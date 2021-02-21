// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package forward_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"mellium.im/xmlstream"
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
