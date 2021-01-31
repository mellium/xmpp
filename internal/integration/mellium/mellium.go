// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mellium facilitates integration testing against clients.
//
// Because the integration package requires starting an external process this
// package must build a testing server into the test binary.
// To enable the server to start the TestMain function must be used in your
// testing package.
// This adds a special flag, -mel.serve, which can be passed to the test binary
// to start the server instead of running the actual tests.
//
// Unlike most servers that we test against, this custom server uses a very
// simple protocol for configuration instead of loading a file:
// On start it receives its configuration encoded as a gob (see
// https://golang.org/pkg/encoding/gob/)  over stdin.
// The tests also go ahead and start listening for connections and pass the
// listeners as extra file descriptors to the server command, this makes sure
// that both sides of the connection know about the listener, its type, its
// ports, etc.
// Finally, the server knows about its temporary directory where it can load
// other files (such as TLS certificates) because it will be started with the
// working directory set to the temp dir.
package mellium // import "mellium.im/xmpp/internal/integration/mellium"

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/internal/integration"
)

type logWriter struct {
	logger *log.Logger
}

func (w logWriter) Write(p []byte) (int, error) {
	w.logger.Printf("%s", p)
	return len(p), nil
}

const (
	serveFlag = "mel.serve"
)

var serve = flag.Bool(serveFlag, false, `internal flag to run a server for use in tests instead of the tests themselves`)

// TestMain must be proxied by any tests that use this package.
// It checks whether the internal -mel.serve flag has been set, and if so
// launches a server to be tested against instead of running the tests.
func TestMain(m *testing.M) {
	flag.Parse()

	if !*serve {
		os.Exit(m.Run())
	}

	logger := log.New(os.Stderr, "server: ", 0)
	wd, err := os.Getwd()
	if err != nil {
		logger.Fatalf("error getting working directory: %v", err)
	}
	logger.Printf("started server in %s", wd)

	dec := gob.NewDecoder(os.Stdin)
	var cfg Config
	err = dec.Decode(&cfg)
	if err != nil {
		logger.Fatalf("error decoding config: %v", err)
	}
	logger.Printf("decoded config: %+v", cfg)

	cfg.C2SFeatures = []xmpp.StreamFeature{
		/* #nosec */
		xmpp.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if info.ServerName == "" {
					info.ServerName = "localhost"
				}
				crt, err := tls.LoadX509KeyPair(info.ServerName+".crt", info.ServerName+".key")
				if err == nil {
					logger.Printf("loaded TLS certificate for %s", info.ServerName)
				}
				return &crt, err
			},
		}),
		xmpp.SASLServer(func(*sasl.Negotiator) bool {
			return true
		}, sasl.Plain,
			sasl.ScramSha256Plus, sasl.ScramSha1Plus,
			sasl.ScramSha256, sasl.ScramSha1),
		xmpp.BindResource(),
	}
	cfg.S2SFeatures = cfg.C2SFeatures

	var c2sListener, s2sListener net.Listener
	var wg sync.WaitGroup
	fdNum := 3
	if cfg.ListenC2S {
		fd := os.NewFile(uintptr(fdNum), "c2sListener")
		defer func() {
			if err := fd.Close(); err != nil {
				logger.Printf("error closing c2s listener file: %v", err)
			}
		}()
		c2sListener, err = net.FileListener(fd)
		if err != nil {
			logger.Fatalf("error opening c2s listener: %v", err)
		}
		fdNum++
		wg.Add(1)
		go func() {
			listen(false, c2sListener, logger, cfg)
			wg.Done()
		}()
	}
	if cfg.ListenS2S {
		fd := os.NewFile(uintptr(fdNum), "s2sListener")
		defer func() {
			if err := fd.Close(); err != nil {
				logger.Printf("error closing s2s listener file: %v", err)
			}
		}()
		s2sListener, err = net.FileListener(fd)
		if err != nil {
			logger.Fatalf("error opening s2s listener: %v", err)
		}
		fdNum++
		wg.Add(1)
		go func() {
			listen(true, s2sListener, logger, cfg)
			wg.Done()
		}()
	}
	go func() {
		s := struct{}{}
		err = dec.Decode(&s)
		if err != nil && err != io.EOF {
			logger.Fatalf("error receiving shutdown signal: %v", err)
		}
		if c2sListener != nil {
			err = c2sListener.Close()
			if err != nil {
				logger.Printf("error closing c2s listener: %v", err)
			}
		}
		if s2sListener != nil {
			err = s2sListener.Close()
			if err != nil {
				logger.Printf("error closing s2s listener: %v", err)
			}
		}
	}()
	wg.Wait()
}

func listen(s2s bool, l net.Listener, logger *log.Logger, cfg Config) {
	connType := "c2s"
	if s2s {
		connType = "s2s"
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Printf("error accepting connection on %s listener: %v", connType, err)
			return
		}
		go func() {
			var mask xmpp.SessionState
			streamCfg := xmpp.StreamConfig{}
			if s2s {
				mask |= xmpp.S2S
				streamCfg.Features = func(*xmpp.Session, ...xmpp.StreamFeature) []xmpp.StreamFeature {
					return cfg.S2SFeatures
				}
			} else {
				streamCfg.Features = func(*xmpp.Session, ...xmpp.StreamFeature) []xmpp.StreamFeature {
					return cfg.C2SFeatures
				}
			}
			if cfg.LogXML {
				streamCfg.TeeIn = logWriter{logger: log.New(logger.Writer(), "RECV ", log.LstdFlags)}
				streamCfg.TeeOut = logWriter{logger: log.New(logger.Writer(), "SEND ", log.LstdFlags)}
			}
			session, err := xmpp.ReceiveSession(context.TODO(), conn, mask, xmpp.NewNegotiator(streamCfg))
			if err != nil {
				logger.Printf("error negotiating %s session: %v", connType, err)
				return
			}
			err = session.Close()
			if err != nil {
				logger.Printf("error closing %s session: %v", connType, err)
			}
		}()
	}
}

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

// ConfigFile is an option that can be used to configure the command.
// Unlike most packages ConfigFile options, this does not actually write the
// config to a file.
func ConfigFile(cfg Config) integration.Option {
	return func(cmd *integration.Cmd) error {
		cmd.Config = cfg
		// We start listening for connections on the testing side so that we know
		// what the port is in advance and can configure it on the cmd.
		// We then pass the listener as an extra file descriptor to the child
		// process (so the first file will be file descriptor 3 as seen from the
		// child process) in the following order:
		// - c2sListener
		// - s2sListener
		//
		// Therefore if no c2s listener is passed, s2s listener will be fd 3,
		// otherwise it will be fd 4.
		if cfg.ListenC2S {
			c2sListener, err := cmd.C2SListen("tcp", ":0")
			if err != nil {
				return err
			}
			fd, err := c2sListener.(interface {
				File() (*os.File, error)
			}).File()
			if err != nil {
				return err
			}
			cmd.Cmd.ExtraFiles = append(cmd.Cmd.ExtraFiles, fd)
		}
		if cfg.ListenS2S {
			s2sListener, err := cmd.S2SListen("tcp", ":0")
			if err != nil {
				return err
			}
			fd, err := s2sListener.(interface {
				File() (*os.File, error)
			}).File()
			if err != nil {
				return err
			}
			cmd.Cmd.ExtraFiles = append(cmd.Cmd.ExtraFiles, fd)
		}
		return nil
	}
}

// Test starts another instance of the tests running in server mode and returns
// a function that runs subtests using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	cmdName, err := os.Executable()
	if err != nil {
		t.Fatalf("could not find testing binary: %v", err)
	}
	opts = append(opts,
		integration.Log(),
		integration.Args("-"+serveFlag),
		integration.Defer(func(cmd *integration.Cmd) error {
			// After the command starts, send its configuration straight to the
			// server over standard input.
			enc := gob.NewEncoder(cmd.Stdin())
			return enc.Encode(getConfig(cmd))
		}),
		integration.Shutdown(func(cmd *integration.Cmd) error {
			return cmd.Stdin().Close()
		}),
	)
	return integration.Test(ctx, cmdName, t, opts...)
}
