package xmpp

import (
	"encoding/xml"
)

type Message struct {
	Stanza
}

func UnmarshalMessage(raw string) (*Message, error) {
	msg := new(Message)
	err := xml.Unmarshal([]byte(raw), &msg)

	return msg, err
}
