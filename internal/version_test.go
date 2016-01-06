// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// Compile time interface checks.
var _ fmt.Stringer = &Version{}
var _ fmt.Stringer = Version{}
var _ xml.MarshalerAttr = &Version{}
var _ xml.MarshalerAttr = Version{}
var _ xml.UnmarshalerAttr = (*Version)(nil)

// Strings must parse correctly.
func TestParseVersion(t *testing.T) {
	for _, data := range []struct {
		vs        string
		v         Version
		shouldErr bool
	}{
		{"1.0", Version{1, 0}, false},
		{"1.0.0", Version{}, true},
		{"1.a", Version{}, true},
		{"1.0xA", Version{}, true},
		{"", Version{}, true},
	} {
		v, err := ParseVersion(data.vs)
		switch {
		case data.shouldErr && err == nil:
			t.Logf("Version '%s' should fail with an error when parsed.", data.vs)
			t.Fail()
		case !data.shouldErr && err != nil:
			t.Logf("Error encountered while parsing '%s': %v", data.vs, err)
			t.Fail()
		case data.shouldErr && err != nil:
			continue
		case !data.shouldErr && err == nil:
			if v != data.v {
				t.Logf("Parsing version %s expected %v but got %v", data.vs, data.v, v)
				t.Fail()
			}
			continue
		}
	}
}
