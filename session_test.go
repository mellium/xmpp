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

type dummyMsg struct {
	Message
	Dummy struct{}
}

type dummyPresence struct {
	Presence
	Dummy struct{}
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

func TestSendDoesNotMutateStanza(t *testing.T) {
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
}

func TestEnforceStanzaSemantics(t *testing.T) {
	t.Run("IQ", func(t *testing.T) {
		s := enforceStanzaSemantics(ping{})
		if s.(ping).ID == "" {
			t.Fatal("Expected ID to be set")
		}
	})
	t.Run("Message", func(t *testing.T) {
		s := enforceStanzaSemantics(dummyMsg{})
		if s.(dummyMsg).ID == "" {
			t.Fatal("Expected ID to be set")
		}
	})
	t.Run("Message", func(t *testing.T) {
		s := enforceStanzaSemantics(dummyPresence{})
		if s.(dummyPresence).ID == "" {
			t.Fatal("Expected ID to be set")
		}
	})
	t.Run("Nonza", func(t *testing.T) {
		var a interface{}
		a = struct{}{}
		b := enforceStanzaSemantics(a)
		if a != b {
			t.Fatal("Expected the same type of nonza to be returned")
		}
	})
}
