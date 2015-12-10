package server

import (
	"crypto/tls"
	"net"
)

// A Server defines parameters for running an XMPP server.
type Server struct {
	options
	handler ConnHandler // Connection handler to invoke.
}

// ListenAndServe listens on the TCP network address srv.ClientAddr and then
// calls Serve to handle requests on incoming connections. If srv.ClientAddr is
// blank, ":xmpp-client" (":5222")is used.
func (srv *Server) ListenAndServe() error {
	clientaddr := srv.ClientAddr
	if clientaddr == "" {
		clientaddr = ":5222"
	}
	ln, err := net.Listen("tcp", srv.ClientAddr)
	if err != nil {
		return err
	}
	return srv.Serve(ln.(*net.TCPListener))
}

// Serve accepts incoming connections on the Listener, spawning a new service
// goroutine for each.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	for {
		rw, e := l.Accept()
		if e != nil {
			// TODO(ssw): Figure out how to handle logging
			continue
		} else {
			go srv.Handler.Handle(rw, l)
		}
	}
}
