// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.
//
// Some Lua embedded in this file is taken from the Prosody Community Modules
// and is licensed under the terms of the MIT license, a copy of which can be
// found in the file "LICENSE.modules".

// Package prosody facilitates integration testing against Prosody.
package prosody // import "mellium.im/xmpp/internal/integration/prosody"

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

// WebSocket enables the websocket module.
// WebSocket implies the HTTPS() option.
func WebSocket() integration.Option {
	return func(cmd *integration.Cmd) error {
		err := Modules("websocket")(cmd)
		if err != nil {
			return err
		}
		return HTTPS()(cmd)
	}
}

// ConfigFile is an option that can be used to write a temporary Prosody config
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
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		return integration.Args(configFlag, cfgFilePath)(cmd)
	}
}

// Ctl returns an option that calls prosodyctl with the provided args.
// It automatically points prosodyctl at the config file so there is no need to
// pass the --config option.
func Ctl(ctx context.Context, args ...string) integration.Option {
	return integration.Defer(ctlFunc(ctx, args...))
}

func ctlFunc(ctx context.Context, args ...string) func(*integration.Cmd) error {
	return func(cmd *integration.Cmd) error {
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		/* #nosec */
		prosodyCtl := exec.CommandContext(ctx, "prosodyctl", configFlag, cfgFilePath)
		prosodyCtl.Args = append(prosodyCtl.Args, args...)
		return prosodyCtl.Run()
	}
}

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

// ListenC2S listens for client-to-server (c2s) connections on a random port.
func ListenC2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		c2sListener, err := cmd.C2SListen("tcp", ":0")
		if err != nil {
			return err
		}
		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle
		// for the TCP connection to it on start, so we're effectively just
		// listening to get a random port that we'll use to configure Prosody, then
		// we need to close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in
		// practice.
		c2sPort := c2sListener.Addr().(*net.TCPAddr).Port
		err = c2sListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.C2SPort = c2sPort
		cmd.Config = cfg
		return nil
	}
}

// ListenS2S listens for server-to-server (s2s) connections on a random port.
func ListenS2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		s2sListener, err := cmd.S2SListen("tcp", "[::1]:0")
		if err != nil {
			return err
		}
		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle for
		// the TCP connection to it on start, so we're effectively just listening to
		// get a random port that we'll use to configure Prosody, then we need to
		// close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in practice.
		s2sPort := s2sListener.Addr().(*net.TCPAddr).Port
		err = s2sListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.S2SPort = s2sPort
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

// MUC launches prosody with the built-in multi-user chat component enabled.
// It is the same as Component(domain, "", "muc", modules).
func MUC(domain string, modules ...string) integration.Option {
	return Component(domain, "", "muc", modules...)
}

// Component adds an component with the given domain and secret to the config
// file.
// If a name is provided the component must be a builtin.
func Component(domain, secret, name string, modules ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		if name == "" {
			compListener, err := cmd.ComponentListen("tcp", "[::1]:0")
			if err != nil {
				return err
			}
			// Prosody creates its own sockets and doesn't provide us with a way of
			// pointing it at an existing Unix domain socket or handing the filehandle
			// for the TCP connection to it on start, so we're effectively just
			// listening to get a random port that we'll use to configure Prosody, then
			// we need to close the connection and let Prosody listen on that port.
			// Technically this is racey, but it's not likely to be a problem in
			// practice.
			compPort := compListener.Addr().(*net.TCPAddr).Port
			err = compListener.Close()
			if err != nil {
				return err
			}

			cfg.CompPort = compPort
		}
		if cfg.Component == nil {
			cfg.Component = make(map[string]struct {
				Name        string
				Secret      string
				Modules     []string
				MUCDefaults []ChannelConfig
			})
		}
		comp := cfg.Component[domain]
		comp.Secret = secret
		comp.Name = name
		comp.Modules = modules
		cfg.Component[domain] = comp
		cmd.Config = cfg
		return nil
	}
}

// HTTPS configures prosody to listen for HTTP and HTTPS on two randomized
// ports and configures TLS certificates for localhost:https.
func HTTPS() integration.Option {
	return func(cmd *integration.Cmd) error {
		httpsListener, err := cmd.HTTPSListen("tcp", "[::1]:0")
		if err != nil {
			return err
		}
		httpListener, err := cmd.HTTPListen("tcp", "[::1]:0")
		if err != nil {
			return err
		}

		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle
		// for the TCP connection to it on start, so we're effectively just
		// listening to get a random port that we'll use to configure Prosody, then
		// we need to close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in
		// practice.
		httpPort := httpListener.Addr().(*net.TCPAddr).Port
		httpsPort := httpsListener.Addr().(*net.TCPAddr).Port
		err = httpListener.Close()
		if err != nil {
			return err
		}
		err = httpsListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.HTTPPort = httpPort
		cfg.HTTPSPort = httpsPort
		cmd.Config = cfg
		return integration.Cert(fmt.Sprintf("localhost:%d", httpsPort))(cmd)
	}
}

// CreateUser returns an option that calls prosodyctl to create a user.
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

// Modules adds custom modules to the enabled modules list.
func Modules(mod ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		cfg.Modules = append(cfg.Modules, mod...)
		cmd.Config = cfg
		return nil
	}
}

// Set adds an extra key/value pair to the global section of the config file.
// If v is a string it will be quoted, otherwise it is marshaled using the %v
// formatting directive (see the fmt package for details).
// As a special case, if v is nil the key is written to the file directly with
// no equals sign.
//
//     -- Set("foo", "bar")
//     foo = "bar"
//
//     -- Set("foo", 123)
//     foo = 123
//
//     -- Set(`Component "conference.example.org" "muc"`, nil)
//     Component "conference.example.org" "muc"
func Set(key string, v interface{}) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		if cfg.Options == nil {
			cfg.Options = make(map[string]interface{})
		}
		cfg.Options[key] = v
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

// Test starts a Prosody instance and returns a function that runs subtests
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig,
		integration.Shutdown(ctlFunc(ctx, "stop")))
	return integration.Test(ctx, cmdName, t, opts...)
}
