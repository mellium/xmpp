// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package integration contains helpers for integration testing.
//
// Normally users writing integration tests should not use this package
// directly, instead they should use the packges in subdirectories of this
// package.
package integration // import "mellium.im/xmpp/internal/integration"

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

// Cmd is an external command being prepared or run.
//
// A Cmd cannot be reused after calling its Run, Output or CombinedOutput
// methods.
type Cmd struct {
	*exec.Cmd

	name          string
	cfgDir        string
	killCtx       context.Context
	kill          context.CancelFunc
	cfgF          func() error
	deferF        func(*Cmd) error
	stdoutWriter  *testWriter
	in, out       *testWriter
	c2sListener   net.Listener
	s2sListener   net.Listener
	compListener  net.Listener
	c2sNetwork    string
	s2sNetwork    string
	httpsListener net.Listener
	httpListener  net.Listener
	httpsNetwork  string
	httpNetwork   string
	compNetwork   string
	shutdown      func(*Cmd) error
	user          jid.JID
	pass          string
	clientCrt     []byte
	clientCrtKey  interface{}
	stdinPipe     io.WriteCloser
	closed        chan error

	// Config is meant to be used by internal packages like prosody and ejabberd
	// to store their internal representation of the config before writing it out.
	Config interface{}
}

// New creates a new, unstarted, command.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, name string, opts ...Option) (*Cmd, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := &Cmd{
		/* #nosec */
		Cmd:          exec.CommandContext(ctx, name),
		name:         name,
		killCtx:      ctx,
		kill:         cancel,
		closed:       make(chan error),
		stdoutWriter: &testWriter{},
	}
	var err error
	cmd.stdinPipe, err = cmd.Cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.cfgDir, err = ioutil.TempDir("", filepath.Base(cmd.name))
	if err != nil {
		return nil, err
	}
	cmd.Cmd.Dir = cmd.cfgDir
	for _, opt := range opts {
		err = opt(cmd)
		if err != nil {
			return nil, fmt.Errorf("error applying option: %v", err)
		}
	}
	if cmd.cfgF != nil {
		err = cmd.cfgF()
		if err != nil {
			return nil, fmt.Errorf("error running config func: %w", err)
		}
	}

	return cmd, nil
}

// Start runs the command.
func (cmd *Cmd) Start() error {
	_, err := fmt.Fprintf(cmd.stdoutWriter, "starting command: %s", cmd)
	if err != nil {
		return err
	}
	err = cmd.Cmd.Start()
	go func() {
		cmd.closed <- cmd.Cmd.Wait()
		close(cmd.closed)
	}()
	return err
}

// Done returns a channel that's closed when the commands process terminates.
func (cmd *Cmd) Done() <-chan error {
	return cmd.closed
}

// Stdin returns a pipe to the commands standard input.
func (cmd *Cmd) Stdin() io.WriteCloser {
	return cmd.stdinPipe
}

// ClientCert returns the last configured client certificate.
// The certificate request info is currently ignored and is only there to make
// promoting this method to a function and using it as
// tls.Config.GetClientCertificate possible.
func (cmd *Cmd) ClientCert(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return &tls.Certificate{
		Certificate: [][]byte{cmd.clientCrt},
		PrivateKey:  cmd.clientCrtKey,
	}, nil
}

// C2SListen returns a listener with a random port.
// The listener is created on the first call to C2SListener.
// Subsequent calls ignore the arguments and return the existing listener.
func (cmd *Cmd) C2SListen(network, addr string) (net.Listener, error) {
	if cmd.c2sListener != nil {
		return cmd.c2sListener, nil
	}

	var err error
	cmd.c2sListener, err = net.Listen(network, addr)
	cmd.c2sNetwork = network
	return cmd.c2sListener, err
}

// S2SListen returns a listener with a random port.
// The listener is created on the first call to S2SListener.
// Subsequent calls ignore the arguments and return the existing listener.
func (cmd *Cmd) S2SListen(network, addr string) (net.Listener, error) {
	if cmd.s2sListener != nil {
		return cmd.s2sListener, nil
	}

	var err error
	cmd.s2sListener, err = net.Listen(network, addr)
	cmd.s2sNetwork = network
	return cmd.s2sListener, err
}

// HTTPSListen returns a listener with a random port (for HTTPS).
// The listener is created on the first call to HTTPSListener.
// Subsequent calls ignore the arguments and return the existing listener.
func (cmd *Cmd) HTTPSListen(network, addr string) (net.Listener, error) {
	if cmd.httpsListener != nil {
		return cmd.httpsListener, nil
	}

	var err error
	cmd.httpsListener, err = net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	cmd.httpsNetwork = network
	return cmd.httpsListener, nil
}

// HTTPListen returns a listener with a random port (for HTTP).
// The listener is created on the first call to HTTPListener.
// Subsequent calls ignore the arguments and return the existing listener.
func (cmd *Cmd) HTTPListen(network, addr string) (net.Listener, error) {
	if cmd.httpListener != nil {
		return cmd.httpListener, nil
	}

	var err error
	cmd.httpListener, err = net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	cmd.httpNetwork = network
	return cmd.httpListener, nil
}

// ComponentListen returns a listener with a random port.
// The listener is created on the first call to ComponentListener.
// Subsequent calls ignore the arguments and return the existing listener.
func (cmd *Cmd) ComponentListen(network, addr string) (net.Listener, error) {
	if cmd.compListener != nil {
		return cmd.compListener, nil
	}

	var err error
	cmd.compListener, err = net.Listen(network, addr)
	cmd.compNetwork = network
	return cmd.compListener, err
}

// ConfigDir returns the temporary directory used to store config files.
func (cmd *Cmd) ConfigDir() string {
	return cmd.cfgDir
}

// Close kills the command if it is still running and cleans up any temporary
// resources that were created.
func (cmd *Cmd) Close() error {
	defer cmd.kill()

	err := cmd.stdinPipe.Close()
	if err != nil {
		return nil
	}

	var e error
	if cmd.shutdown != nil {
		e = cmd.shutdown(cmd)
	}
	ctx, cancel := context.WithTimeout(cmd.killCtx, 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return fmt.Errorf("command did not exit in time: %v", ctx.Err())
	case err = <-cmd.closed:
		if err != nil {
			return fmt.Errorf("error waiting on command to exit: %v", err)
		}
	}
	if err != nil {
		return fmt.Errorf("error waiting on command to exit: %v", err)
	}
	err = os.RemoveAll(cmd.cfgDir)
	if err != nil {
		return err
	}
	return e
}

// User returns the address and password of a user created on the server (if
// any).
func (cmd *Cmd) User() (jid.JID, string) {
	return cmd.user, cmd.pass
}

// DialClient attempts to connect to the server with a client-to-server (c2s)
// connection by dialing the address reserved by C2SListen and then negotiating
// a stream with the location set to the domainpart of j and the origin set to
// j.
func (cmd *Cmd) DialClient(ctx context.Context, j jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	return cmd.dial(ctx, false, j.Domain(), j, t, features...)
}

// DialServer attempts to connect to the server with a server-to-server (s2s)
// connection by dialing the address reserved by S2SListen and then negotiating
// a stream.
func (cmd *Cmd) DialServer(ctx context.Context, location, origin jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	return cmd.dial(ctx, true, location, origin, t, features...)
}

// C2SAddr returns the client-to-server address and network.
func (cmd *Cmd) C2SAddr() (net.Addr, string) {
	return cmd.c2sListener.Addr(), cmd.c2sNetwork
}

// C2SPort returns the port on which the C2S listener is running (if any).
func (cmd *Cmd) C2SPort() string {
	addr, _ := cmd.C2SAddr()
	/* #nosec */
	_, port, _ := net.SplitHostPort(addr.String())
	return port
}

// HTTPSPort returns the port on which the HTTPS listener is running (if any).
func (cmd *Cmd) HTTPSPort() string {
	addr := cmd.httpsListener.Addr()
	/* #nosec */
	_, port, _ := net.SplitHostPort(addr.String())
	return port
}

// HTTPPort returns the port on which the HTTP listener is running (if any).
func (cmd *Cmd) HTTPPort() string {
	addr := cmd.httpListener.Addr()
	/* #nosec */
	_, port, _ := net.SplitHostPort(addr.String())
	return port
}

// S2SAddr returns the server-to-server address and network.
func (cmd *Cmd) S2SAddr() (net.Addr, string) {
	return cmd.s2sListener.Addr(), cmd.s2sNetwork
}

// ComponentAddr returns the component address and network.
func (cmd *Cmd) ComponentAddr() (net.Addr, string) {
	return cmd.compListener.Addr(), cmd.compNetwork
}

// ComponentConn dials a connection to the component socket and returns it
// without negotiating a session.
func (cmd *Cmd) ComponentConn(ctx context.Context) (net.Conn, error) {
	if cmd.compListener == nil {
		return nil, errors.New("component not configured, please configure a component listener")
	}
	addr := cmd.compListener.Addr().String()
	network := cmd.compNetwork

	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("error dialing %s: %w", addr, err)
	}
	return conn, nil
}

// Conn dials a connection and returns it without negotiating a session.
func (cmd *Cmd) Conn(ctx context.Context, s2s bool) (net.Conn, error) {
	switch {
	case s2s && cmd.s2sListener == nil:
		return nil, errors.New("s2s not configured, please configure an s2s listener")
	case !s2s && cmd.c2sListener == nil:
		return nil, errors.New("c2s not configured, please configure a c2s listener")
	}

	var addr, network string
	if s2s {
		addr = cmd.s2sListener.Addr().String()
		network = cmd.s2sNetwork
	} else {
		addr = cmd.c2sListener.Addr().String()
		network = cmd.c2sNetwork
	}

	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("error dialing %s: %w", addr, err)
	}
	return conn, nil
}

// HTTPConn dials a connection and returns it without negotiating a session.
func (cmd *Cmd) HTTPConn(ctx context.Context, secure bool) (net.Conn, error) {
	switch {
	case secure && cmd.httpsListener == nil:
		return nil, errors.New("HTTPS not configured, please configure an HTTPS listener")
	case !secure && cmd.httpListener == nil:
		return nil, errors.New("HTTP not configured, please configure an HTTP listener")
	}

	var addr, network string
	if secure {
		addr = cmd.httpsListener.Addr().String()
		network = cmd.httpsNetwork
	} else {
		addr = cmd.httpListener.Addr().String()
		network = cmd.httpNetwork
	}

	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("error dialing %s: %w", addr, err)
	}
	return conn, nil
}

func (cmd *Cmd) dial(ctx context.Context, s2s bool, location, origin jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	conn, err := cmd.Conn(ctx, s2s)
	if err != nil {
		return nil, err
	}
	negotiator := xmpp.NewNegotiator(func(*xmpp.Session, xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Features: features,
			TeeIn:    cmd.in,
			TeeOut:   cmd.out,
		}
	})
	var mask xmpp.SessionState
	if s2s {
		mask |= xmpp.S2S
	}
	session, err := xmpp.NewSession(
		ctx,
		location,
		origin,
		conn,
		mask,
		negotiator,
	)
	if err != nil {
		return nil, fmt.Errorf("error establishing session: %w", err)
	}
	return session, nil
}

// Option is used to configure a Cmd.
type Option func(cmd *Cmd) error

// User sets the values that will be returned by a call to cmd.User later. It
// does not actually create a user.
func User(user jid.JID, pass string) Option {
	return func(cmd *Cmd) error {
		cmd.user = user
		cmd.pass = pass
		return nil
	}
}

// Shutdown is run before the configuration is removed and is meant to
// gracefully shutdown the application in case it does not handle the kill
// signal correctly.
// If multiple shutdown options are used the functions will be run in the order
// in which they are passed until an error is encountered.
func Shutdown(f func(*Cmd) error) Option {
	return func(cmd *Cmd) error {
		if cmd.shutdown != nil {
			prev := cmd.shutdown
			cmd.shutdown = func(cmd *Cmd) error {
				err := prev(cmd)
				if err != nil {
					return err
				}
				return f(cmd)
			}
			return nil
		}
		cmd.shutdown = f
		return nil
	}
}

// Args sets additional command line args to be passed to the command.
func Args(f ...string) Option {
	return func(cmd *Cmd) error {
		cmd.Cmd.Args = append(cmd.Args, f...)
		return nil
	}
}

// Cert creates a private key and certificate with the given name.
func Cert(name string) Option {
	return cert(name, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		DNSNames:     []string{filepath.Base(name)},
	})
}

// ClientCert creates a private key and certificate with the given name that
// can be used for TLS authentication.
func ClientCert(name string) Option {
	return cert(name, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		DNSNames:     []string{filepath.Base(name)},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
}

func cert(name string, crt *x509.Certificate) Option {
	return func(cmd *Cmd) error {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		err = TempFile(name+".key", func(_ *Cmd, w io.Writer) error {
			return pem.Encode(w, &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			})
		})(cmd)
		if err != nil {
			return err
		}
		return TempFile(name+".crt", func(_ *Cmd, w io.Writer) error {
			cert, err := x509.CreateCertificate(rand.Reader, crt, crt, key.Public(), key)
			if err != nil {
				return err
			}
			if len(crt.ExtKeyUsage) > 0 && crt.ExtKeyUsage[0] == x509.ExtKeyUsageClientAuth {
				cmd.clientCrt = cert
				cmd.clientCrtKey = key
			}
			return pem.Encode(w, &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert,
			})
		})(cmd)
	}
}

// TempFile creates a file in the commands temporary working directory.
// After all configuration is complete it then calls f to populate the config
// files.
func TempFile(cfgFileName string, f func(*Cmd, io.Writer) error) Option {
	return func(cmd *Cmd) (err error) {
		dir := filepath.Dir(cfgFileName)
		if dir != "" && dir != "." && dir != "/" && dir != ".." {
			err = os.MkdirAll(filepath.Join(cmd.cfgDir, dir), 0700)
			if err != nil {
				return err
			}
		}

		newF := func() error {
			cfgFilePath := filepath.Join(cmd.cfgDir, cfgFileName)
			/* #nosec */
			cfgFile, err := os.OpenFile(cfgFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}

			err = f(cmd, cfgFile)
			if err != nil {
				/* #nosec */
				cfgFile.Close()
				return err
			}
			return cfgFile.Close()
		}
		if cmd.cfgF != nil {
			prev := cmd.cfgF
			cmd.cfgF = func() error {
				err := prev()
				if err != nil {
					return err
				}
				return newF()
			}
			return nil
		}
		cmd.cfgF = newF
		return nil
	}
}

type testWriter struct {
	sync.Mutex
	t   *testing.T
	tag string
}

func (w *testWriter) Write(p []byte) (int, error) {
	if w == nil {
		return len(p), nil
	}
	w.Lock()
	defer w.Unlock()

	if w.t != nil {
		w.t.Logf("%s%s", w.tag, p)
	}
	return len(p), nil
}

func (w *testWriter) Update(t *testing.T) {
	if w == nil {
		return
	}
	w.Lock()
	w.t = t
	w.Unlock()
}

// Log configures the command to copy stdout to the current testing.T.
// This should not be used for CLI or TUI clients.
func Log() Option {
	return func(cmd *Cmd) error {
		cmd.Cmd.Stdout = cmd.stdoutWriter
		cmd.Cmd.Stderr = cmd.stdoutWriter
		return nil
	}
}

// LogFile reads the provided file into the log in the same way that the Log
// option reads a commands standard output.
// It can optionally copy the command to the provided io.Writer (if non-nil) as
// well similar to the tee(1) command.
// Normally this should be used by the various server and client packages to
// read a FIFO which has been configured to be the log by the actual client
// process so that it functions similar to the tail(1) command.
func LogFile(f string, tee io.Writer) Option {
	return Defer(func(cmd *Cmd) error {
		var r io.Reader
		/* #nosec */
		fd, err := os.OpenFile(f, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			return err
		}
		if tee != nil {
			r = io.TeeReader(fd, tee)
		} else {
			r = fd
		}
		go func() {
			for {
				_, err := io.Copy(cmd.stdoutWriter, r)
				if err != nil {
					fmt.Fprintf(cmd.stdoutWriter, "error copying log file to stdout: %v", err)
					return
				}
			}
		}()
		return nil
	})
}

// LogXML configures the command to log sent and received XML to the current
// testing.T.
func LogXML() Option {
	return func(cmd *Cmd) error {
		cmd.in = &testWriter{tag: "RECV"}
		cmd.out = &testWriter{tag: "SENT"}
		return nil
	}
}

// Defer is an option that calls f after the command is started.
// If multiple Defer options are passed they are called in order until an error
// is encountered.
func Defer(f func(*Cmd) error) Option {
	return func(cmd *Cmd) error {
		if cmd.deferF != nil {
			prev := cmd.deferF
			cmd.deferF = func(cmd *Cmd) error {
				err := prev(cmd)
				if err != nil {
					return err
				}
				return f(cmd)
			}
			return nil
		}
		cmd.deferF = f
		return nil
	}
}

// Test starts a command and returns a function that runs tests as a subtest
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, name string, t *testing.T, opts ...Option) SubtestRunner {
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	_, err := exec.LookPath(name)
	if err != nil {
		i := -1
		return func(f func(context.Context, *testing.T, *Cmd)) bool {
			i++
			return t.Run(fmt.Sprintf("%s/%d", filepath.Base(name), i), func(t *testing.T) {
				t.Skip(err.Error())
			})
		}
	}

	cmd, err := New(ctx, name, opts...)
	if err != nil {
		t.Fatalf("error creating command: %v", err)
	}

	t.Cleanup(func() {
		err := cmd.Close()
		if err != nil {
			t.Logf("error cleaning up test: %v", err)
		}
	})
	cmd.stdoutWriter.Update(t)
	cmd.in.Update(t)
	cmd.out.Update(t)
	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}
	if cmd.c2sListener != nil {
		err = waitSocket(cmd.c2sNetwork, cmd.c2sListener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
	}
	if cmd.s2sListener != nil {
		err = waitSocket(cmd.s2sNetwork, cmd.s2sListener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
	}
	if cmd.deferF != nil {
		err = cmd.deferF(cmd)
		if err != nil {
			t.Fatal(err)
		}
	}

	i := -1
	return func(f func(context.Context, *testing.T, *Cmd)) bool {
		i++
		return t.Run(fmt.Sprintf("%s/%d", filepath.Base(name), i), func(t *testing.T) {
			if tw, ok := cmd.Cmd.Stdout.(*testWriter); ok {
				tw.Update(t)
			}
			cmd.in.Update(t)
			cmd.out.Update(t)
			f(ctx, t, cmd)
		})
	}
}

// SubtestRunner is the signature of a function that can be used to start
// subtests.
type SubtestRunner func(func(context.Context, *testing.T, *Cmd)) bool
