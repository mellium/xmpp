// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package mcabber facilitates integration testing against Mcabber.
package mcabber // import "mellium.im/xmpp/internal/integration/mcabber"

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName = "mcabberrc"
	cmdName     = "mcabber"
	configFlag  = "-f"
	controlFIFO = "command.socket"
)

// Send transmits the given command over the control pipe.
func Send(cmd *integration.Cmd, s string) error {
	cfg := getConfig(cmd)
	_, err := fmt.Fprintln(cfg.FIFO, s)
	return err
}

// Ping sends an XMPP ping through Mcabber.
func Ping(cmd *integration.Cmd, to jid.JID) error {
	return Send(cmd, fmt.Sprintf("request ping %s", to))
}

// ConfigFile is an option that can be used to write a temporary config file.
// This will overwrite the existing config file and make most of the other
// options in this package noops.
// This option only exists for the rare occasion that you need complete control
// over the config file.
func ConfigFile(cfg Config) integration.Option {
	return func(cmd *integration.Cmd) error {
		if cfg.FIFO == nil {
			fifoPath := filepath.Join(cmd.ConfigDir(), controlFIFO)
			err := unix.Mkfifo(fifoPath, 0660)
			if err != nil {
				return err
			}
			cfg.FIFO, err = os.OpenFile(fifoPath, os.O_RDWR, os.ModeNamedPipe)
			if err != nil {
				return err
			}
		}

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

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	return integration.Shutdown(func(cmd *integration.Cmd) error {
		return Send(cmd, "quit")
	})(cmd)
}

// Test starts a Mcabber instance and returns a function that runs subtests
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return integration.Test(ctx, cmdName, t, opts...)
}
