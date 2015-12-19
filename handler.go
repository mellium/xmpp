package xmpp

import (
	"encoding/xml"
)

type Handler interface {
	Handle(encoder xml.Encoder, decoder xml.Decoder) error
}
