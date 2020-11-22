// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ejabberd facilitates integration testing against Ejabberd.
package ejabberd // import "mellium.im/xmpp/internal/integration/ejabberd"

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"testing"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName = "ejabberd.yml"
	cmdName     = "ejabberdctl"
	configFlag  = "--config-dir"
	logsFlag    = "--logs"
	spoolFlag   = "--spool"
)

// New creates a new, unstarted, ejabberd daemon.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error) {
	opts = append(opts, foreground)
	cmd, err := integration.New(
		ctx, cmdName,
		opts...,
	)
	return cmd, err
}

// ConfigFile is an option that can be used to write a temporary Ejabberd config
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
		err = integration.Args(configFlag, cmd.ConfigDir())(cmd)
		if err != nil {
			return err
		}
		err = integration.Args(logsFlag, cmd.ConfigDir())(cmd)
		if err != nil {
			return err
		}
		return integration.Args(spoolFlag, cmd.ConfigDir())(cmd)
	}
}

func defaultConfig(cmd *integration.Cmd) error {
	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	c2sListener, err := cmd.C2SListen("unix", filepath.Join(cmd.ConfigDir(), "c2s.socket"))
	if err != nil {
		return err
	}
	c2sSocket := c2sListener.Addr().(*net.UnixAddr).Name

	s2sListener, err := cmd.S2SListen("unix", filepath.Join(cmd.ConfigDir(), "s2s.socket"))
	if err != nil {
		return err
	}
	s2sSocket := s2sListener.Addr().(*net.UnixAddr).Name

	// The config file didn't exist, so create a default config.
	return ConfigFile(Config{
		VHosts:    []string{"localhost"},
		C2SSocket: c2sSocket,
		S2SSocket: s2sSocket,
	})(cmd)
}

func inetrcFile(cmd *integration.Cmd) error {
	return integration.TempFile("inetrc", func(_ *integration.Cmd, w io.Writer) error {
		_, err := fmt.Fprint(w, inetrc)
		return err
	})(cmd)
}

func foreground(cmd *integration.Cmd) error {
	return integration.Args("foreground")(cmd)
}

// Ctl returns an option that calls ejabberdctl with the provided args after the
// config has been written.
// It automatically points ejabberdctl at the config file path.
func Ctl(ctx context.Context, args ...string) integration.Option {
	return integration.Defer(ctlFunc(ctx, args...))
}

func ctlFunc(ctx context.Context, args ...string) func(*integration.Cmd) error {
	return func(cmd *integration.Cmd) error {
		cfgFilePath := cmd.ConfigDir()
		/* #nosec */
		ejabberdCtl := exec.CommandContext(ctx, "ejabberdctl",
			configFlag, cfgFilePath, logsFlag, cfgFilePath, spoolFlag, cfgFilePath)
		ejabberdCtl.Args = append(ejabberdCtl.Args, args...)
		return ejabberdCtl.Run()
	}
}

// CreateUser returns an option that calls ejabberdctl to create a user.
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

// Test starts an Ejabberd instance and returns a function that runs f as a
// subtest using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig, inetrcFile, foreground,
		integration.Shutdown(ctlFunc(ctx, "stop")))
	return integration.Test(ctx, cmdName, t, opts...)
}
