// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package cid_test

import (
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"

	"mellium.im/xmpp/internal/cid"
)

const invalidURL = "%gh&%ij"

var parseTestCases = []struct {
	in       string
	out      string
	hashName string
	domain   string
	hash     string
	errText  string
}{
	0: {
		in:       "cid:name+abcd@domain",
		hashName: "name",
		domain:   "domain",
		hash:     "abcd",
	},
	1: {
		in:       "n+ab@d",
		out:      "cid:n+ab@d",
		hashName: "n",
		domain:   "d",
		hash:     "ab",
	},
	2: {
		in:      "cid:+abcd@domain",
		errText: "cid: missing hash name",
	},
	3: {
		in:      "cid:abcd@domain",
		errText: "cid: missing hash name",
	},
	4: {
		in:      "cid:name+@domain",
		errText: "cid: no hash found",
	},
	5: {
		in:      "cid:name+ab@",
		errText: "cid: missing domain part",
	},
	6: {
		in:      "cid:name+ab",
		errText: "cid: missing domain part",
	},
	7: {
		in:      "cid:wat",
		errText: "cid: missing domain part",
	},
	8: {
		in:      "cid:sha1+wat@bob.xmpp.org",
		errText: "cid: error decoding hash: encoding/hex: invalid byte: U+0077 'w'",
	},
	9: {
		in:      "https://example.com",
		errText: `cid: failed to parse URL with scheme "https"`,
	},
	10: {
		in:      "cid://test:123",
		errText: `cid: URL is invalid and resulted in empty CID`,
	},
	11: {
		in: invalidURL,
		errText: func() string {
			//lint:ignore SA1007 deliberately parsing an invalid URL to test against
			_, err := url.Parse(invalidURL)
			return err.Error()
		}(),
	},
}

func TestParse(t *testing.T) {
	for i, tc := range parseTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			u, err := cid.Parse(tc.in)
			if tc.errText != "" && err == nil {
				t.Fatalf("expected error during parsing")
			}
			if tc.errText != "" {
				if err.Error() != tc.errText {
					t.Fatalf("unexpected error text parsing CID: want=%v, got=%v", tc.errText, err.Error())
				}
				return
			}

			if u.HashName != tc.hashName {
				t.Errorf("wrong hash name: want=%v, got=%v", tc.hashName, u.HashName)
			}
			if u.Domain != tc.domain {
				t.Errorf("wrong domain: want=%v, got=%v", tc.domain, u.Domain)
			}
			if hash := hex.EncodeToString(u.Hash); hash != tc.hash {
				t.Errorf("wrong hash: want=%v, got=%v", tc.hash, hash)
			}
			out := tc.out
			if out == "" {
				out = tc.in
			}
			if s := u.String(); s != out {
				t.Errorf("wrong string output: want=%v, got=%v", out, s)
			}
		})
	}
}
