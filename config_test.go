// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"testing"
)

// New configs should populate the features map with no duplicates.
func TestNewConfigShouldPopulateFeatures(t *testing.T) {
	c := NewServerConfig(nil, nil, BindResource(), BindResource(), StartTLS(true))
	if len(c.Features) != 2 {
		t.Errorf("Expected two features (Bind and StartTLS) but got: %v", c.Features)
	}
}
