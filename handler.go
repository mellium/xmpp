package xmpp

type Handler interface {
	Handle(encoder xml.Encoder, decoder xml.Decoder) error
}
