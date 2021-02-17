// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package paging_test

import (
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/paging"
)

var (
	_ xmlstream.Marshaler = (*paging.RequestCount)(nil)
	_ xmlstream.WriterTo  = (*paging.RequestCount)(nil)
	_ xml.Marshaler       = (*paging.RequestCount)(nil)
	_ xmlstream.Marshaler = (*paging.RequestNext)(nil)
	_ xmlstream.WriterTo  = (*paging.RequestNext)(nil)
	_ xml.Marshaler       = (*paging.RequestNext)(nil)
	_ xmlstream.Marshaler = (*paging.RequestPrev)(nil)
	_ xmlstream.WriterTo  = (*paging.RequestPrev)(nil)
	_ xml.Marshaler       = (*paging.RequestPrev)(nil)
	_ xmlstream.Marshaler = (*paging.RequestIndex)(nil)
	_ xmlstream.WriterTo  = (*paging.RequestIndex)(nil)
	_ xml.Marshaler       = (*paging.RequestIndex)(nil)
	_ xmlstream.Marshaler = (*paging.Set)(nil)
	_ xmlstream.WriterTo  = (*paging.Set)(nil)
	_ xml.Marshaler       = (*paging.Set)(nil)
)

var iterTests = [...]struct {
	in          string
	out         string
	nextQueries string
	prevQueries string
	curQueries  string
	err         error
}{
	0: {
		in: `<a></a>`,
	},
	1: {
		in:  `<nums><a>1</a><a/></nums>`,
		out: `<a>1</a><a></a>`,
	},
	2: {
		in: `<nums><a>1</a><b/><set xmlns='http://jabber.org/protocol/rsm'>
<last>2</last>
</set>
</nums>`,
		out:         "<a>1</a><b></b>\n",
		nextQueries: `<set xmlns="http://jabber.org/protocol/rsm"><max>10</max><after>2</after></set>`,
		curQueries:  `<set xmlns="http://jabber.org/protocol/rsm"><first></first><last>2</last></set>`,
	},
	3: {
		in: `<nums><set xmlns='http://jabber.org/protocol/rsm'>
<first>1</first>
</set><b/></nums>`,
		out:         "<b></b>",
		prevQueries: `<set xmlns="http://jabber.org/protocol/rsm"><before>1</before><max>10</max></set>`,
		curQueries:  `<set xmlns="http://jabber.org/protocol/rsm"><first>1</first><last></last></set>`,
	},
}

func TestIter(t *testing.T) {
	for i, tc := range iterTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var buf, curQueries, nextQueries, prevQueries strings.Builder
			d := xml.NewDecoder(strings.NewReader(tc.in))
			e := xml.NewEncoder(&buf)
			_, err := d.Token()
			if err != nil {
				t.Fatalf("error popping first token: %v", err)
			}
			iter := paging.NewIter(d, 10)
			nextSet := iter.NextPage()
			if nextSet != nil {
				t.Fatalf("should not start with next page set, got %+v", nextSet)
			}
			for iter.Next() {
				start, r := iter.Current()
				if start != nil {
					err := e.EncodeToken(*start)
					if err != nil {
						t.Fatalf("error encoding start element: %v", err)
					}
				}
				_, err = xmlstream.Copy(e, r)
				if err != nil {
					t.Fatalf("error encoding stream: %v", err)
				}
			}
			if err := iter.Err(); err != nil {
				t.Fatalf("error iterating: %v", err)
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("error flushing output: %v", err)
			}
			// Next
			query, err := xml.Marshal(iter.NextPage())
			if err != nil {
				t.Fatalf("error marshaling next set: %v", err)
			}
			_, err = nextQueries.Write(query)
			if err != nil {
				t.Fatalf("error writing next query: %v", err)
			}
			// Prev
			query, err = xml.Marshal(iter.PreviousPage())
			if err != nil {
				t.Fatalf("error marshaling previous set: %v", err)
			}
			_, err = prevQueries.Write(query)
			if err != nil {
				t.Fatalf("error writing prev query: %v", err)
			}
			// Current
			query, err = xml.Marshal(iter.CurrentPage())
			if err != nil {
				t.Fatalf("error marshaling current set: %v", err)
			}
			_, err = curQueries.Write(query)
			if err != nil {
				t.Fatalf("error writing current query: %v", err)
			}
			if out := buf.String(); out != tc.out {
				t.Errorf("wrong output: want=%s, got=%s", tc.out, out)
			}
			if q := nextQueries.String(); q != tc.nextQueries {
				t.Errorf("wrong next queries:\nwant=%s,\n got=%s", tc.nextQueries, q)
			}
			if q := prevQueries.String(); q != tc.prevQueries {
				t.Errorf("wrong prev queries:\nwant=%s,\n got=%s", tc.prevQueries, q)
			}
			if q := curQueries.String(); q != tc.curQueries {
				t.Errorf("wrong current queries:\nwant=%s,\n got=%s", tc.curQueries, q)
			}
		})
	}
}
