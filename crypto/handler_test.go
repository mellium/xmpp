// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package crypto_test

import (
	_ "crypto/sha256"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/disco/info"
)

var iterTests = []struct {
	node string
	h    []crypto.Hash
	vars []string
	err  error
}{
	0: {
		h:    []crypto.Hash{crypto.SHA256, crypto.BLAKE2b_512},
		vars: []string{"urn:xmpp:hash-function-text-names:sha-256"},
		err:  crypto.ErrUnlinkedAlgo,
	},
	1: {
		node: "test",
		h:    []crypto.Hash{crypto.SHA256},
	},
}

func TestIter(t *testing.T) {
	for i, tc := range iterTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			iter := crypto.Features(tc.h...)
			var found []string
			err := iter.ForFeatures(tc.node, func(feature info.Feature) error {
				found = append(found, feature.Var)
				return nil
			})
			if len(found) != len(tc.vars) {
				t.Fatalf("wrong length for iter: want=%d, got=%d", len(tc.vars), len(found))
			}
			if !reflect.DeepEqual(tc.vars, found) {
				t.Errorf("found incorrect hashes: want=%v, got=%v", tc.h, found)
			}
			if tc.err != nil || err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("expected unlinked algo error, found: %v", err)
				}
			}
		})
	}
}
