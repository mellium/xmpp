package server

import "net"

type Handler interface {
	Handle(c net.Conn, l net.Listener) error
}
