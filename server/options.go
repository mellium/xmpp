package server

type Option func(*options)
type options struct {
	clientAddr string // TCP address to listen on, ":xmpp-client" if empty.
	tlsConfig  *tls.Config
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}
	return
}

// The ClientAddr option sets the interface and port that the server will listen
// on for inbounc connections from XMPP clients.
func ClientAddr(string addr) Option {
	return func(o *options) {
		o.clientAddr = addr
	}
}

// The TLSConfig option fully configures the servers TLS including the
// certificate chains used, cipher suites, etc. based on the given tls.Config.
func TLSConfig(config *tls.Config) {
	return func(o *options) {
		o.tlsConfig = config
	}
}

var (
	PreferClientCipherSuites = preferClientCipherSuites // Prefer cipher suite order indicated by the client (not recommended).
)

var preferServerCipherSuites = func(o *options) {
	if o.tlsConfig == nil {
		o.tlsConfig = &tls.Config{}
	}
	o.tlsConfig.PreferServerCipherSuites = true
}
