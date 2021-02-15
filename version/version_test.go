// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package version_test

import (
	"context"
	"encoding/xml"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/version"
)

var (
	_ xmlstream.Marshaler = (*version.Query)(nil)
	_ xmlstream.WriterTo  = (*version.Query)(nil)
)

var marshalTests = [...]struct {
	in  version.Query
	out string
}{
	0: {
		in:  version.Query{},
		out: `<query xmlns="` + version.NS + `"></query>`,
	},
	1: {
		in: version.Query{
			Name:    "name",
			Version: "ver",
			OS:      "os",
		},
		out: `<query xmlns="` + version.NS + `"><name>name</name><version>ver</version><os>os</os></query>`,
	},
}

func TestMarshal(t *testing.T) {
	for i, tc := range marshalTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Run("marshal", func(t *testing.T) {
				b, err := xml.Marshal(tc.in)
				if err != nil {
					t.Fatalf("error marshaling IQ: %v", err)
				}
				if string(b) != tc.out {
					t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, b)
				}
			})
			t.Run("encode", func(t *testing.T) {
				var buf strings.Builder
				e := xml.NewEncoder(&buf)
				_, err := tc.in.WriteXML(e)
				if err != nil {
					t.Fatalf("error writing XML: %v", err)
				}
				if err = e.Flush(); err != nil {
					t.Fatalf("error flushing XML: %v", err)
				}
				if out := buf.String(); out != tc.out {
					t.Errorf("wrong output:\nwant=%s,\n got=%s", tc.out, out)
				}
			})
		})
	}
}

func TestGet(t *testing.T) {
	query := version.Query{
		Name:    "name",
		Version: "ver",
		OS:      "os",
	}
	cs := xmpptest.NewClientServer(
		xmpptest.ServerHandlerFunc(func(e xmlstream.TokenReadEncoder, start *xml.StartElement) error {
			iq, err := stanza.NewIQ(*start)
			if err != nil {
				return err
			}
			_, err = xmlstream.Copy(e, iq.Result(query.TokenReader()))
			return err
		}),
	)
	resp, err := version.Get(context.Background(), cs.Client, jid.JID{})
	if err != nil {
		t.Fatalf("error querying version: %v", err)
	}
	expectedName := xml.Name{Space: version.NS, Local: "query"}
	if resp.XMLName != expectedName {
		t.Errorf("wrong XML name: want=%v, got=%v", expectedName, resp.XMLName)
	}
	resp.XMLName = xml.Name{}
	if !reflect.DeepEqual(resp, query) {
		t.Errorf("unexpected response: want=%v, got=%v", query, resp)
	}
}
