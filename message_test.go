// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmpp/jid"
)

// TODO: Make this a table test and add some more complicated messages.
// TODO: How should we test marshalling? Probably don't want to assume that
//       attribute order will remain stable.

func TestDefaults(t *testing.T) {
	var mt messageType

	if mt != NormalMessage {
		t.Log("Default value of message type should be 'normal'.")
		t.Fail()
	}
}

// Tests unmarshalling of a single XML blob into a message.
func TestUnmarshalMessage(t *testing.T) {
	mb := []byte(`
<message
    from='juliet@example.com/balcony'
    id='ktx72v49'
    to='romeo@example.net'
    type='chat'
    xml:lang='en'>
  <body>Art thou not Romeo, and a Montague?</body>
</message>
	`)
	m := &Message{
		stanza{
			To:   &jid.JID{},
			From: &jid.JID{},
		},
		xml.Name{},
	}
	err := xml.Unmarshal(mb, m)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if m.stanza.To.String() != "romeo@example.net" {
		t.Logf("Expected %s but got %s", "romeo@example.net", m.stanza.To.String())
		t.Fail()
	}
	if m.stanza.To.String() != "romeo@example.net" {
		t.Logf("Expected %s but got %s", "romeo@example.net", m.stanza.To.String())
		t.Fail()
	}
	if m.stanza.ID != "ktx72v49" {
		t.Logf("Expected %s but got %s", "ktx72v49", m.stanza.To.String())
		t.Fail()
	}
}

// Messages must be marshalable to XML
func TestMarshalMessage(t *testing.T) {
	j := jid.MustParse("feste@shakespeare.lit")
	m := Message{
		stanza{
			ID:   "1234",
			To:   j,
			Lang: "en",
		},
		xml.Name{Space: "jabber:client", Local: "message"},
	}
	// TODO: Check the output; is the order guaranteed?
	if _, err := xml.Marshal(m); err != nil {
		t.Log(err)
		t.Fail()
	}
}
