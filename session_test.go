package xmpp

import (
	"bytes"
	"encoding/xml"
	"testing"
)

// ping is an XEP-0199 ping
type ping struct {
	IQ
	Ping struct{} `xml:"urn:xmpp:ping ping"`
}

func newDummySession() (*bytes.Buffer, *Session) {
	b := new(bytes.Buffer)
	s := &Session{
		rw:         b,
		features:   make(map[string]interface{}),
		negotiated: make(map[string]struct{}),
	}
	s.out.e = xml.NewEncoder(b)
	return b, s
}

func TestSendEnforcesIQSemantics(t *testing.T) {
	_, s := newDummySession()
	p := &ping{}
	err := s.Send(p)
	if err != nil {
		t.Fatal(err)
	}
	p2 := ping{}
	if *p != p2 {
		t.Fatalf("Sending mutated original struct")
	}
	// TODO: Test to make sure we set the ID
}
