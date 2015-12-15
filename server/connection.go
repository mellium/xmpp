package server

type Handler interface {
	Handle(c net.Conn, l net.Listener) error
}
