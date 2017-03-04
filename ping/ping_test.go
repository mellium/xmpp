// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package ping

import (
	"encoding/xml"
	"testing"
)

func TestMarshal(t *testing.T) {
	p := Ping{}
	b, err := xml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	out := `<iq id="" type="get"><ping xmlns="urn:xmpp:ping"></ping></iq>`
	// TODO: This is probably flakey because the order of the id/type attributes
	//       isn't guaranteed.
	if string(b) != out {
		t.Errorf("Marshaled invalid ping, want=`%s`, got=`%s`", out, b)
	}
}
