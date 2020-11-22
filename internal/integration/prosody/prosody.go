// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package prosody facilitates integration testing against Prosody.
package prosody // import "mellium.im/xmpp/internal/integration/prosody"

import (
	"context"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"testing"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName = "prosody.cfg.lua"
	cmdName     = "prosody"
	configFlag  = "--config"
)

// New creates a new, unstarted, prosody daemon.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error) {
	return integration.New(
		ctx, cmdName,
		opts...,
	)
}

// ConfigFile is an option that can be used to write a temporary Prosody config
// file.
func ConfigFile(cfg Config) integration.Option {
	return func(cmd *integration.Cmd) error {
		err := integration.TempFile(cfgFileName, func(cmd *integration.Cmd, w io.Writer) error {
			return cfgTmpl.Execute(w, struct {
				Config
				ConfigDir string
			}{
				Config:    cfg,
				ConfigDir: cmd.ConfigDir(),
			})
		})(cmd)
		if err != nil {
			return err
		}
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		return integration.Args(configFlag, cfgFilePath)(cmd)
	}
}

// Ctl returns an option that calls prosodyctl with the provided args.
// It automatically points prosodyctl at the config file so there is no need to
// pass the --config option.
func Ctl(ctx context.Context, args ...string) integration.Option {
	return integration.Defer(func(cmd *integration.Cmd) error {
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		/* #nosec */
		prosodyCtl := exec.CommandContext(ctx, "prosodyctl", configFlag, cfgFilePath)
		prosodyCtl.Args = append(prosodyCtl.Args, args...)
		err := prosodyCtl.Run()
		return err
	})
}

// CreateUser returns an option that calls prosodyctl to create a user.
// It is equivalent to calling:
// Ctl(ctx, "register", "localpart", "domainpart", "password").
func CreateUser(ctx context.Context, addr, pass string) integration.Option {
	return func(cmd *integration.Cmd) error {
		j, err := jid.Parse(addr)
		if err != nil {
			return err
		}
		return Ctl(ctx, "register", j.Localpart(), j.Domainpart(), pass)(cmd)
	}
}

func defaultConfig(cmd *integration.Cmd) error {
	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}
	c2sListener, err := cmd.C2SListen("tcp", "[::1]:0")
	if err != nil {
		return err
	}
	// Prosody creates its own sockets and doesn't provide us with a way of
	// pointing it at an existing Unix domain socket or handing the filehandle for
	// the TCP connection to it on start, so we're effectively just listening to
	// get a random port that we'll use to configure Prosody, then we need to
	// close the connection and let Prosody listen on that port.
	// Technically this is racey, but it's not likely to be a problem in practice.
	defer c2sListener.Close()

	s2sListener, err := cmd.S2SListen("tcp", "[::1]:0")
	if err != nil {
		return err
	}
	defer s2sListener.Close()

	// The config file didn't exist, so create a default config.
	return ConfigFile(Config{
		VHosts:  []string{"localhost"},
		C2SPort: c2sListener.Addr().(*net.TCPAddr).Port,
		S2SPort: s2sListener.Addr().(*net.TCPAddr).Port,
	})(cmd)
}

// Test starts a Prosody instance and returns a function that runs subtests
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return integration.Test(ctx, cmdName, t, opts...)
}
