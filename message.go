package xmpp

import (
	"encoding/xml"
)

type Message struct {
	stanza
}

func UnmarshalMessage(raw string) (*Message, error) {
	msg := new(Message)
	err := xml.Unmarshal([]byte(raw), &msg)

	return msg, err
}
