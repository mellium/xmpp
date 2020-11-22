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

	name        string
	cfgDir      string
	kill        context.CancelFunc
	cfgF        []func() error
	deferF      []func(*Cmd) error
	in, out     *testWriter
	c2sListener net.Listener
	s2sListener net.Listener
	c2sNetwork  string
	s2sNetwork  string
	shutdown    func(*Cmd) error
}

// New creates a new, unstarted, command.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, name string, opts ...Option) (*Cmd, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := &Cmd{
		/* #nosec */
		Cmd:  exec.CommandContext(ctx, name),
		name: name,
		kill: cancel,
	}
	for _, opt := range opts {
		err := opt(cmd)
		if err != nil {
			return nil, err
		}
	}
	for _, f := range cmd.cfgF {
		err := f()
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
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

// ConfigDir returns the temporary directory used to store config files.
func (cmd *Cmd) ConfigDir() string {
	return cmd.cfgDir
}

// Close kills the command if it is still running and cleans up any temporary
// resources that were created.
func (cmd *Cmd) Close() error {
	defer cmd.kill()

	var e error
	if cmd.shutdown != nil {
		e = cmd.shutdown(cmd)
	}
	if cmd.cfgDir != "" {
		err := os.RemoveAll(cmd.cfgDir)
		if err != nil {
			return err
		}
	}
	return e
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

func (cmd *Cmd) dial(ctx context.Context, s2s bool, location, origin jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	switch {
	case s2s && cmd.s2sListener == nil:
		return nil, errors.New("s2s not configured, please call S2SListen first")
	case !s2s && cmd.c2sListener == nil:
		return nil, errors.New("c2s not configured, please call C2SListen first")
	}

	addr := cmd.c2sListener.Addr().String()
	network := cmd.c2sNetwork
	if s2s {
		addr = cmd.s2sListener.Addr().String()
		network = cmd.s2sNetwork
	}

	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("error dialing %s: %w", addr, err)
	}
	negotiator := xmpp.NewNegotiator(xmpp.StreamConfig{
		Features: features,
		TeeIn:    cmd.in,
		TeeOut:   cmd.out,
	})
	session, err := xmpp.NegotiateSession(
		ctx,
		location,
		origin,
		conn,
		false,
		negotiator,
	)
	if err != nil {
		return nil, fmt.Errorf("error establishing session: %w", err)
	}
	return session, nil
}

// Option is used to configure a Cmd.
type Option func(cmd *Cmd) error

// Shutdown is run before the configuration is removed and is meant to
// gracefully shutdown the application in case it does not handle the kill
// signal correctly.
func Shutdown(f func(*Cmd) error) Option {
	return func(cmd *Cmd) error {
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
			crt := &x509.Certificate{
				SerialNumber: big.NewInt(1),
				NotBefore:    time.Now(),
				NotAfter:     time.Now().Add(365 * 24 * time.Hour),
				DNSNames:     []string{filepath.Base(name)},
			}
			cert, err := x509.CreateCertificate(rand.Reader, crt, crt, key.Public(), key)
			if err != nil {
				return err
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
		if cmd.cfgDir == "" {
			cmd.cfgDir, err = ioutil.TempDir("", cmd.name)
			if err != nil {
				return err
			}
		}
		dir := filepath.Dir(cfgFileName)
		if dir != "" && dir != "." && dir != "/" && dir != ".." {
			err = os.MkdirAll(filepath.Join(cmd.cfgDir, dir), 0700)
			if err != nil {
				return err
			}
		}

		cmd.cfgF = append(cmd.cfgF, func() error {
			cfgFilePath := filepath.Join(cmd.cfgDir, cfgFileName)
			cfgFile, err := os.Create(cfgFilePath)
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
		})
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

// Log configures the command to log output to the current testing.T.
func Log() Option {
	return func(cmd *Cmd) error {
		w := &testWriter{}
		cmd.Cmd.Stdout = w
		cmd.Cmd.Stderr = w
		return nil
	}
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

// Defer is an option that calls f the command is started.
func Defer(f func(*Cmd) error) Option {
	return func(cmd *Cmd) error {
		cmd.deferF = append(cmd.deferF, f)
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

	cmd, err := New(ctx, name, opts...)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := cmd.Close()
		if err != nil {
			t.Logf("error cleaning up test: %v", err)
		}
	})

	if tw, ok := cmd.Cmd.Stdout.(*testWriter); ok {
		tw.Update(t)
	}
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
	for _, f := range cmd.deferF {
		err := f(cmd)
		if err != nil {
			t.Fatal(err)
		}
	}

	i := -1
	return func(f func(context.Context, *testing.T, *Cmd)) bool {
		i++
		return t.Run(fmt.Sprintf("%s/%d", name, i), func(t *testing.T) {
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
