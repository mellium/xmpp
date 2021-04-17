// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"encoding/xml"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/stream"
	"mellium.im/xmpp/internal/xmpptest"
	streamerr "mellium.im/xmpp/stream"
)

var readerTestCases = [...]struct {
	in        string
	skip      int
	err       error
	errType   error
	errStrCmp bool
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
		err: streamerr.Error{},
	},
	8: {
		in:  `<error xmlns='http://etherx.jabber.org/streams'/>`,
		err: streamerr.Error{},
	},
	9: {
		in: `<stream:error xmlns:stream='http://etherx.jabber.org/streams'>
	<not-well-formed xmlns='urn:ietf:params:xml:ns:xmpp-streams'/>
</stream:error>`,
		err: streamerr.NotWellFormed,
	},
	10: {
		in: `<stream:error xmlns:stream='http://etherx.jabber.org/streams'>
	</notwellformed>
</stream:error>`,
		errType: &xml.SyntaxError{},
	},
	11: {
		in:   `<stream:stream xmlns:stream='http://etherx.jabber.org/streams'></stream:stream>`,
		skip: 1,
	},
	12: {
		in:        `<message><!-- Test --></message></stream:stream>`,
		err:       errors.New("invalid token type: xml.Comment"),
		errStrCmp: true,
	},
	13: {
		in:        `<iq><!dir></iq></stream:stream>`,
		err:       errors.New("invalid token type: xml.Directive"),
		errStrCmp: true,
	},
	14: {
		in:        `<iq><!dir></iq>`,
		err:       errors.New("invalid token type: xml.Directive"),
		errStrCmp: true,
	},
	15: {
		in:        `<iq><?xml?></iq>`,
		err:       errors.New("invalid token type: xml.ProcInst"),
		errStrCmp: true,
	},
	16: {
		// Chardata is checked elsewhere because it is valid as long as it's not at
		// the top level, or is at the top level but is only whitespace and this
		// reader doesn't track the nesting level.
		in: `aaa<iq></iq>`,
	},
}

func TestReader(t *testing.T) {
	for i, tc := range readerTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var out strings.Builder
			d := xml.NewDecoder(strings.NewReader(tc.in))
			e := xml.NewEncoder(&out)
			for tc.skip > 0 {
				// Throw away any tokens that we need to make the stream well-formed,
				// but that we don't want in the actual test output.
				tok, err := d.Token()
				if err != nil {
					t.Fatalf("error skipping tokens: %v", err)
				}
				err = e.EncodeToken(tok)
				if err != nil {
					t.Fatalf("error encoding skipped tokens: %v", err)
				}
				tc.skip--
			}
			_, err := xmlstream.Copy(e, stream.Reader(d))
			switch {
			case tc.errStrCmp:
				if err == nil {
					err = errors.New("nil")
				}
				if tc.err == nil {
					tc.err = errors.New("nil")
				}
				if tc.err.Error() != err.Error() {
					t.Errorf("unexpected error string: want=%q, got=%q", tc.err, err)
				}
			case tc.errType != nil:
				if reflect.TypeOf(tc.errType) != reflect.TypeOf(err) {
					t.Errorf("unexpected error type: want=%T, got=%T", tc.err, err)
				}
			case !errors.Is(err, tc.err):
				t.Errorf("unexpected error: want=%v, got=%#v", tc.err, err)
			}
			if err = e.Flush(); err != nil {
				t.Fatalf("error flushing output to buffer: %v", err)
			}
			t.Logf("output: %q", out.String())
		})
	}
}

func TestBadFormat(t *testing.T) {
	toks := &xmpptest.Tokens{
		xml.EndElement{Name: xml.Name{Local: "error", Space: streamerr.NS}},
	}
	r := stream.Reader(toks)
	var out strings.Builder
	e := xml.NewEncoder(&out)
	_, err := xmlstream.Copy(e, r)
	if !errors.Is(err, streamerr.BadFormat) {
		t.Errorf("unexpected error: want=%v, got=%v", streamerr.BadFormat, err)
	}
	err = e.Flush()
	if err != nil {
		t.Errorf("error flushing: %v", err)
	}
	t.Logf("output: %q", out.String())
}
