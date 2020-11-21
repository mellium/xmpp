// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/stream"
	streamerr "mellium.im/xmpp/stream"
)

var readerTestCases = [...]struct {
	in  string
	err error
}{
	0: {},
	1: {
		in: `<stream></stream>`,
	},
	2: {
		in: `<stream:stream
					version='1.0'
					xmlns='jabber:client'
					xmlns:stream='http://wrong.namespace.example.org/'/>`,
	},
	3: {
		in: `<other:stream
					version='1.0'
					xmlns='jabber:client'
					xmlns:other='http://etherx.jabber.org/streams'/>`,
		err: stream.ErrUnexpectedRestart,
	},
	4: {
		in: `<stream:stream
					version='1.0'
					xmlns='jabber:client'
					xmlns:stream='http://etherx.jabber.org/streams'/>`,
		err: stream.ErrUnexpectedRestart,
	},
	5: {
		in: `<stream:unknown
					version='1.0'
					xmlns='jabber:client'
					xmlns:stream='http://etherx.jabber.org/streams'/>`,
		err: stream.ErrUnknownStreamElement,
	},
	6: {
		in: `<stream:error/>`,
	},
	7: {
		in:  `<stream:error xmlns:stream='http://etherx.jabber.org/streams'/>`,
		err: streamerr.InternalServerError,
	},
}

func TestReader(t *testing.T) {
	for i, tc := range readerTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var out strings.Builder
			d := xml.NewDecoder(strings.NewReader(tc.in))
			e := xml.NewEncoder(&out)
			_, err := xmlstream.Copy(e, stream.Reader(d))
			if err != tc.err {
				t.Errorf("unexpected error: want=%v, got=%v", tc.err, err)
			}
			if err = e.Flush(); err != nil {
				t.Fatalf("error flushing output to buffer: %v", err)
			}
			t.Logf("output: %q", out.String())
		})
	}
}
