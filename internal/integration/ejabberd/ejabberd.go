// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ejabberd facilitates integration testing against Ejabberd.
package ejabberd // import "mellium.im/xmpp/internal/integration/ejabberd"

import (
	"context"
	"fmt"
	"io"
	"testing"

	"mellium.im/xmpp/internal/integration"
)

const (
	cfgFileName = "ejabberd.yml"
	configFlag  = "--config-dir"
	cmdName     = "ejabberdctl"
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
		err = integration.Args("--logs", cmd.ConfigDir())(cmd)
		if err != nil {
			return err
		}
		return integration.Args("--spool", cmd.ConfigDir())(cmd)
	}
}

func defaultConfig(cmd *integration.Cmd) error {
	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	// The config file didn't exist, so create a default config.
	return ConfigFile(Config{
		VHosts: []string{"localhost"},
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

// Test starts an Ejabberd instance and returns a function that runs f as a
// subtest using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig, inetrcFile, foreground)
	return integration.Test(ctx, cmdName, t, opts...)
}
