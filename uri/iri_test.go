// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package uri_test

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/uri"
)

var parseTests = [...]struct {
	raw    string
	iri    string
	u      *uri.URI
	err    error
	values url.Values
}{
	0: {err: uri.TestErrBadScheme},
	1: {raw: "mailto:badscheme@example.net", err: uri.TestErrBadScheme},
	2: {
		raw: "xmpp:feste@example.net",
		u:   &uri.URI{ToAddr: jid.MustParse("feste@example.net")},
	},
	3: {
		raw: "xmpp://feste@example.net",
		u:   &uri.URI{AuthAddr: jid.MustParse("feste@example.net")},
	},
	4: {
		raw: "xmpp:feste@example.net/ilyria",
		u:   &uri.URI{ToAddr: jid.MustParse("feste@example.net/ilyria")},
	},
	5: {
		raw: "xmpp://feste@example.net/olivia@example.org",
		u: &uri.URI{
			AuthAddr: jid.MustParse("feste@example.net"),
			ToAddr:   jid.MustParse("olivia@example.org"),
		},
	},
	6: {
		raw: "xmpp:example-node@example.com?message",
		u: &uri.URI{
			ToAddr: jid.MustParse("example-node@example.com"),
			Action: "message",
		},
		values: url.Values{
			"message": []string{""},
		},
	},
	7: {
		raw: "xmpp:example-node@example.com?message;subject=Hello%20World",
		iri: "xmpp:example-node@example.com?message;subject=Hello World",
		u: &uri.URI{
			ToAddr: jid.MustParse("example-node@example.com"),
			Action: "message",
		},
		values: url.Values{
			"message": []string{""},
			"subject": []string{"Hello World"},
		},
	},
	8: {
		raw: "xmpp:example-node@example.com?message&subject=Hello%20World",
		iri: "xmpp:example-node@example.com?message&subject=Hello World",
		u: &uri.URI{
			ToAddr: jid.MustParse("example-node@example.com"),
			Action: "message",
		},
		values: url.Values{
			"message": []string{""},
			"subject": []string{"Hello World"},
		},
	},
	9: {
		// Tests that JID errors in the xmpp: parsing path are passed through.
		raw: "xmpp:feste@/ilyria",
		err: jidErr("feste@/ilyria"),
	},
	10: {
		// Tests that JID errors in the xmpp:// parsing path are passed through.
		raw: "xmpp://b&d@example.net",
		err: jidErr("b&d@example.net"),
	},
	11: {
		// Tests that recipient JID errors in the xmpp:// parsing path are passed
		// through.
		raw: "xmpp://feste@example.net/b&d@example.net",
		err: jidErr("b&d@example.net"),
	},
	12: {
		raw: "xmpp://nasty!%23$%25()*+,-.;=%3F%5B%5C%5D%5E_%60%7B%7C%7D~node@example.com/node@example.com/repulsive%20!%23%22$%25&'()*+,-.%2F:;%3C=%3E%3F%40%5B%5C%5D%5E_%60%7B%7C%7D~resource",
		iri: "xmpp://nasty!#$%()*+,-.;=?[\\]^_`{|}~node@example.com/node@example.com/repulsive !#\"$%&'()*+,-./:;<=>?@[\\]^_`{|}~resource",
		u: &uri.URI{
			AuthAddr: jid.MustParse("nasty!#$%()*+,-.;=?[\\]^_`{|}~node@example.com"),
			ToAddr:   jid.MustParse("node@example.com/repulsive !#\"$%&'()*+,-./:;<=>?@[\\]^_`{|}~resource"),
		},
	},
	13: {
		raw: "xmpp:node@example.com/repulsive%20!%23%22$%25&'()*+,-.%2F:;%3C=%3E%3F%40%5B%5C%5D%5E_%60%7B%7C%7D~resource",
		iri: "xmpp:node@example.com/repulsive !#\"$%&'()*+,-./:;<=>?@[\\]^_`{|}~resource",
		u: &uri.URI{
			ToAddr: jid.MustParse("node@example.com/repulsive !#\"$%&'()*+,-./:;<=>?@[\\]^_`{|}~resource"),
		},
	},
	14: {
		// Errors from unescaping the path should be passed through.
		raw: "xmpp:test%%bad",
		err: func() error {
			_, err := url.PathUnescape("xmpp:%%b")
			return err
		}(),
	},
	15: {
		// Test from RFC 3987 §3.2.1
		raw: "xmpp:example.org/D%C3%BCrst",
		iri: "xmpp:example.org/Dürst",
		u: &uri.URI{
			ToAddr: jid.MustParse("example.org/Dürst"),
		},
	},
	16: {
		// Test from RFC 3987 §3.2.1
		raw: "xmpp:example.org/D%FCrst",
		iri: "xmpp:example.org/D%FCrst",
		u: &uri.URI{
			ToAddr: jid.MustParse("example.org/D%FCrst"),
		},
	},
	17: {
		// Test from RFC 3987 §3.2.1
		raw: "xmpp:xn--99zt52a.example.org/%e2%80%ae",
		iri: "xmpp:xn--99zt52a.example.org/%E2%80%AE",
		u: &uri.URI{
			ToAddr: jid.MustParse("\u7D0D\u8C46.example.org/%E2%80%AE"),
		},
	},
	18: {
		// Check that allowed multi-byte scalar values get written even if the fast
		// path is skipped.
		raw: "xmpp:feste@example.net/%E2%80☃",
		iri: "xmpp:feste@example.net/%E2%80☃",
		u: &uri.URI{
			ToAddr: jid.MustParse("feste@example.net/%E2%80☃"),
		},
	},
}

func TestParse(t *testing.T) {
	for i, tc := range parseTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			u, err := uri.Parse(tc.raw)
			if !errors.Is(err, tc.err) {
				t.Fatalf("unexpected error: want=%v, got=%v", tc.err, err)
			}
			if err != nil {
				return
			}

			// Check the output of the IRI method.
			// As a shortcut, if the expected IRI is the empty string check against
			// the raw string (nothing in the URI needs to be decoded, so IRI()
			// should return the original).
			if tc.iri == "" {
				tc.iri = tc.raw
			}
			if iri := u.String(); tc.iri != iri {
				t.Errorf("unexpected IRI decoded: want=%q, got=%q", tc.iri, iri)
			}

			if !u.AuthAddr.Equal(tc.u.AuthAddr) {
				t.Errorf("unexpected auth address: want=%v, got=%v", tc.u.AuthAddr, u.AuthAddr)
			}

			if !u.ToAddr.Equal(tc.u.ToAddr) {
				t.Errorf("unexpected recipient address: want=%v, got=%v", tc.u.ToAddr, u.ToAddr)
			}

			if u.Action != tc.u.Action {
				t.Errorf("wrong action: want=%v, got=%v", tc.u.Action, u.Action)
			}

			if tc.values == nil {
				tc.values = make(map[string][]string)
			}
			if values := u.URL.Query(); !reflect.DeepEqual(tc.values, values) {
				t.Errorf("unexpected query values:want=%v, got=%v", tc.values, values)
			}
		})
	}
}

func TestParseErrorsReturned(t *testing.T) {
	const badURL = " xmpp://feste@example.net"
	_, err := uri.Parse(badURL)
	//lint:ignore SA1007 Deliberately testing parsing of invalid URL
	_, expected := url.Parse(badURL)
	if expected == nil {
		t.Fatal("test requires an invalid URL")
	}
	if err.Error() != expected.Error() {
		t.Fatalf("expected URL parse errors to be returned: want=%v, got=%v", expected, err)
	}
}

// jidErr should be passed a bad JID. It is used to extract the internal error
// returned by the JID package so that it can be compared against parse errors
// that should return either the error from JID parsing, or a new error that
// wraps it.
// If the provided JID would not result in an error, jidErr panics to avoid
// regressions where the tests aren't actually comparing the error we expected.
func jidErr(addr string) error {
	_, err := jid.Parse(addr)
	if err == nil {
		panic("test requires bad data, but no error was encountered parsing JID")
	}
	return err
}

func BenchmarkParse(b *testing.B) {
	for i, tc := range parseTests {
		b.Run(strconv.Itoa(i), func(b *testing.B) {
			if tc.err != nil {
				b.Skip("don't benchmark tests that would error")
			}

			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, _ = uri.Parse(tc.raw)
			}
		})
	}
}
