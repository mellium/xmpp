package xmpp

import (
	"bitbucket.org/SamWhited/go-jid"
	"testing"
)

const STREAM = `
<stream:stream
    from='juliet@im.example.com'
    to='im.example.com'
    version='1.0'
    xml:lang='en'
    xmlns='jabber:client'
    xmlns:stream='http://etherx.jabber.org/streams'>
`

func TestStreamProperties(t *testing.T) {
	stream, err := NewStream(STREAM)
	if err != nil {
		t.FailNow()
	}

	// Test From
	f, err := stream.From()
	if err != nil {
		t.FailNow()
	}
	if from, err := jid.NewJID("juliet@im.example.com"); err != nil || !f.Equals(from) {
		t.FailNow()
	}

	// Test To
	to, err := stream.To()
	if err != nil {
		t.FailNow()
	}
	if to2, err := jid.NewJID("im.example.com"); err != nil || !to.Equals(to2) {
		t.FailNow()
	}
}
