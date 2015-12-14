package server

import (
	"net"
)

// A Server defines parameters for running an XMPP server.
type Server struct {
	options
	// handler ConnHandler // Connection handler to invoke.
}

// New creates a new XMPP server with the given options.
func New(opts ...Option) *Server {
	return &Server{
		options: getOpts(opts...),
	}
}

// ListenAndServe listens on the TCP network address ClientAddr and then calls
// Serve to handle requests on incoming connections. If ClientAddr is blank,
// ":xmpp-client" (":5222")is used.
func (srv *Server) ListenAndServe() error {
	clientaddr := srv.options.clientAddr
	if clientaddr == "" {
		clientaddr = ":5222"
	}
	ln, err := net.Listen("tcp", srv.options.clientAddr)
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
		_, e := l.Accept()
		if e != nil {
			// TODO(ssw): Figure out how to handle logging
			continue
		} else {
			// go srv.Handler.Handle(rw, l)
		}
	}
}
