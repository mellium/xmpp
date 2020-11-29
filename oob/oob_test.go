// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package oob_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/oob"
	"mellium.im/xmpp/stanza"
)

var (
	_ xmlstream.WriterTo  = oob.IQ{}
	_ xmlstream.Marshaler = oob.IQ{}
	_ xmlstream.WriterTo  = oob.Query{}
	_ xmlstream.Marshaler = oob.Query{}
	_ xmlstream.WriterTo  = oob.Data{}
	_ xmlstream.Marshaler = oob.Data{}
)

func TestEncodeIQ(t *testing.T) {
	j := jid.MustParse("feste@example.net")

	iq := oob.IQ{
		IQ: stanza.IQ{To: j},
		Query: oob.Query{
			URL:  "url",
			Desc: "desc",
		},
	}

	t.Run("marshal", func(t *testing.T) {
		out, err := xml.Marshal(iq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		const expected = `<iq id="" to="feste@example.net" from="" type="get"><query xmlns="jabber:iq:oob"><url>url</url><desc>desc</desc></query></iq>`
		if string(out) != expected {
			t.Errorf("wrong encoding:\nwant=%s,\n got=%s", expected, out)
		}
	})

	t.Run("write", func(t *testing.T) {
		var b strings.Builder
		e := xml.NewEncoder(&b)
		_, err := iq.WriteXML(e)
		if err != nil {
			t.Fatalf("error writing XML token stream: %v", err)
		}
		err = e.Flush()
		if err != nil {
			t.Fatalf("error flushing token stream: %v", err)
		}

		const expected = `<iq type="" to="feste@example.net"><query xmlns="jabber:iq:oob"><url>url</url><desc>desc</desc></query></iq>`
		if streamOut := b.String(); streamOut != expected {
			t.Errorf("wrong stream encoding:\nwant=%s,\n got=%s", expected, streamOut)
		}
	})
}

func TestEncodeData(t *testing.T) {
	data := oob.Data{
		URL:  "url",
		Desc: "desc",
	}

	t.Run("marshal", func(t *testing.T) {
		out, err := xml.Marshal(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		const expected = `<x xmlns="jabber:x:oob"><url>url</url><desc>desc</desc></x>`
		if string(out) != expected {
			t.Errorf("wrong encoding:\nwant=%s,\n got=%s", expected, out)
		}
	})

	t.Run("write", func(t *testing.T) {
		var b strings.Builder
		e := xml.NewEncoder(&b)
		_, err := data.WriteXML(e)
		if err != nil {
			t.Fatalf("error writing XML token stream: %v", err)
		}
		err = e.Flush()
		if err != nil {
			t.Fatalf("error flushing token stream: %v", err)
		}

		const expected = `<x xmlns="jabber:x:oob"><url>url</url><desc>desc</desc></x>`
		if streamOut := b.String(); streamOut != expected {
			t.Errorf("wrong stream encoding:\nwant=%s,\n got=%s", expected, streamOut)
		}
	})
}

func TestEncodeQuery(t *testing.T) {
	// This test both ensures that Query's WriteXML method is tested (it is not
	// used when encoding an IQ that contains a query) and makes sure that desc is
	// optional but URL is not.
	query := oob.Query{}

	t.Run("marshal", func(t *testing.T) {
		out, err := xml.Marshal(query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		const expected = `<query xmlns="jabber:iq:oob"><url></url></query>`
		if string(out) != expected {
			t.Errorf("wrong encoding:\nwant=%s,\n got=%s", expected, out)
		}
	})

	t.Run("write", func(t *testing.T) {
		var b strings.Builder
		e := xml.NewEncoder(&b)
		_, err := query.WriteXML(e)
		if err != nil {
			t.Fatalf("error writing XML token stream: %v", err)
		}
		err = e.Flush()
		if err != nil {
			t.Fatalf("error flushing token stream: %v", err)
		}

		const expected = `<query xmlns="jabber:iq:oob"><url></url></query>`
		if streamOut := b.String(); streamOut != expected {
			t.Errorf("wrong stream encoding:\nwant=%s,\n got=%s", expected, streamOut)
		}
	})
}
