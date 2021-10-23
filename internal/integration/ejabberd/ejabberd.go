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
// This will overwrite the existing config file and make most of the other
// options in this package noops.
// This option only exists for the rare occasion that you need complete control
// over the config file.
func ConfigFile(cfg Config) integration.Option {
	return func(cmd *integration.Cmd) error {
		cmd.Config = cfg
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

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

// ListenC2S listens for client-to-server (c2s) connections on a Unix domain
// socket.
func ListenC2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		c2sListener, err := cmd.C2SListen("unix", filepath.Join(cmd.ConfigDir(), "c2s.socket"))
		if err != nil {
			return err
		}
		c2sSocket := c2sListener.Addr().(*net.UnixAddr).Name
		err = c2sListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.C2SSocket = c2sSocket
		cmd.Config = cfg
		return nil
	}
}

// ListenS2S listens for server-to-server (s2s) connections on a Unix domain
// socket.
func ListenS2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		s2sListener, err := cmd.S2SListen("unix", filepath.Join(cmd.ConfigDir(), "s2s.socket"))
		if err != nil {
			return err
		}
		s2sSocket := s2sListener.Addr().(*net.UnixAddr).Name

		cfg := getConfig(cmd)
		cfg.S2SSocket = s2sSocket
		cmd.Config = cfg
		return nil
	}
}

// VHost configures one or more virtual hosts.
// The default if this option is not provided is to create a single vhost called
// "localhost" and create a self-signed cert for it (if VHost is specified certs
// must be manually created).
func VHost(hosts ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		cfg.VHosts = append(cfg.VHosts, hosts...)
		cmd.Config = cfg
		return nil
	}
}

func defaultConfig(cmd *integration.Cmd) error {
	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	cfg := getConfig(cmd)
	if len(cfg.VHosts) == 0 {
		const vhost = "localhost"
		cfg.VHosts = append(cfg.VHosts, vhost)
		err := integration.Cert(vhost)(cmd)
		if err != nil {
			return err
		}
	}
	cmd.Config = cfg
	if j, _ := cmd.User(); j.Equal(jid.JID{}) {
		err := CreateUser(context.TODO(), "me@"+cfg.VHosts[0], "password")(cmd)
		if err != nil {
			return err
		}
	}

	return ConfigFile(cfg)(cmd)
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

// Component adds an external component with the given domain and secret to the
// config file.
func Component(domain, secret string) integration.Option {
	return func(cmd *integration.Cmd) error {
		compListener, err := cmd.ComponentListen("unix", filepath.Join(cmd.ConfigDir(), "comp.socket"))
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.CompSocket = compListener.Addr().(*net.UnixAddr).Name
		if cfg.Component == nil {
			cfg.Component = make(map[string]string)
		}
		cfg.Component[domain] = secret
		cmd.Config = cfg
		return nil
	}
}

// WebSocket listens for WebSocket connections.
func WebSocket() integration.Option {
	return func(cmd *integration.Cmd) error {
		httpsListener, err := cmd.HTTPSListen("unix", filepath.Join(cmd.ConfigDir(), "https.socket"))
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.HTTPSocket = httpsListener.Addr().(*net.UnixAddr).Name
		cmd.Config = cfg
		return nil
	}
}

// CreateUser returns an option that calls ejabberdctl to create a user.
// It is equivalent to calling:
// Ctl(ctx, "register", "localpart", "domainpart", "password") except that it
// also configures the underlying Cmd to know about the user.
func CreateUser(ctx context.Context, addr, pass string) integration.Option {
	return func(cmd *integration.Cmd) error {
		j, err := jid.Parse(addr)
		if err != nil {
			return err
		}
		err = Ctl(ctx, "register", j.Localpart(), j.Domainpart(), pass)(cmd)
		if err != nil {
			return err
		}
		return integration.User(j, pass)(cmd)
	}
}

// Test starts an Ejabberd instance and returns a function that runs f as a
// subtest using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig, inetrcFile, foreground,
		integration.Shutdown(ctlFunc(ctx, "stop")),
		integration.Shutdown(ctlFunc(ctx, "stopped")),
	)
	return integration.Test(ctx, cmdName, t, opts...)
}
