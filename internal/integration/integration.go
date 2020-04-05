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
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

// Cmd is an external command being prepared or run.
//
// A Cmd cannot be reused after calling its Run, Output or CombinedOutput
// methods.
type Cmd struct {
	*exec.Cmd

	name    string
	cfgDir  string
	kill    context.CancelFunc
	cfgF    []func() error
	in, out *testWriter
}

// New creates a new, unstarted, command.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, name string, opts ...Option) (*Cmd, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := &Cmd{
		Cmd:  exec.CommandContext(ctx, name),
		name: name,
		kill: cancel,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	for _, f := range cmd.cfgF {
		err := f()
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

// ConfigDir returns the temporary directory used to store config files.
func (cmd *Cmd) ConfigDir() string {
	return cmd.cfgDir
}

// Close kills the command if it is still running and cleans up any temporary
// resources that were created.
func (cmd *Cmd) Close() error {
	defer cmd.kill()

	if cmd.cfgDir != "" {
		err := os.RemoveAll(cmd.cfgDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Dial attempts to connect to the server by dialing localhost and then
// negotiating a stream with the location set to the domainpart of j and the
// origin set to j.
func (cmd *Cmd) Dial(ctx context.Context, j jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error) {
	dialer := dial.Dialer{
		NoLookup: true,
		NoTLS:    true,
	}
	conn, err := dialer.Dial(ctx, "tcp", jid.MustParse("localhost"))
	if err != nil {
		return nil, fmt.Errorf("error dialing %s: %w", j, err)
	}
	negotiator := xmpp.NewNegotiator(xmpp.StreamConfig{
		Features: features,
		TeeIn:    cmd.in,
		TeeOut:   cmd.out,
	})
	session, err := xmpp.NegotiateSession(
		ctx,
		j.Domain(),
		j,
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

			defer cfgFile.Close()
			return f(cmd, cfgFile)
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

// Test starts a command and returns a function that runs f as a subtest using
// t.Run.
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
	err = waitPort()
	if err != nil {
		t.Fatal(err)
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
