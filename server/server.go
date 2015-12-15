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

// ListenAndServeC2S listens on the TCP network address ClientAddr and then
// calls ServeC2S to handle requests on incoming connections. If ClientAddr is
// blank, ":xmpp-client" (":5222") is used.
func (srv *Server) ListenAndServe() error {
	clientaddr := srv.options.clientAddr
	if clientaddr == "" {
		clientaddr = ":5222"
	}
	ln, err := net.Listen("tcp", clientaddr)
	if err != nil {
		return err
	}
	return srv.ServeC2S(ln.(*net.TCPListener))
}

// ServeC2S accepts incoming connections on the Listener, spawning a new C2S
// service goroutine for each.
func (srv *Server) ServeC2S(l net.Listener) (err error) {
	defer func() {
		if cerr := l.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	for {
		c, e := l.Accept()
		if e != nil {
			continue
		} else {
			go func() {
				session := &C2SSession{}
				session.Handle(c, l)
			}()
		}
	}
}
