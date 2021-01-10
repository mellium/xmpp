// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package sendxmpp facilitates integration testing with sendxmpp.
package sendxmpp // import "mellium.im/xmpp/internal/integration/sendxmpp"

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName = "sendxmpprc"
	cmdName     = "sendxmpp"
	configFlag  = "-f"
)

// Send transmits the given message (or xml, if the Raw option was used) over
// sendxmpp.
func Send(cmd *integration.Cmd, s string) error {
	stdin := cmd.Stdin()
	_, err := io.WriteString(stdin, s)
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdin, "\n")
	return err
}

// Ping sends an XMPP ping.
func Ping(cmd *integration.Cmd, to jid.JID) error {
	return Send(cmd, fmt.Sprintf(
		`<iq to="%s" id="123" type="get"><ping xmlns='urn:xmpp:ping'/></iq>`, to,
	))
}

// New creates a new, unstarted, sendxmpp running as a daemon using interactive
// mode.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error) {
	return integration.New(
		ctx, cmdName,
		opts...,
	)
}

// ConfigFile is an option that can be used to write a temporary sendxmpp config
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

// Raw configures sendxmpp to send raw XML instead of messages.
func Raw() integration.Option {
	return integration.Args("--raw")
}

// TLS configures sendxmpp to log in with TLS.
func TLS() integration.Option {
	return integration.Args("--tls", "--no-tls-verify")
}

// Debug enables debug logging for sendxmpp (logging must still be enabled using
// integration.Log).
// Using it multiple times increases the log level.
func Debug() integration.Option {
	return integration.Args("-d")
}

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	err := integration.Args("-v", "-i", "-r", "sendxmpp")(cmd)
	if err != nil {
		return err
	}

	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	cfg := getConfig(cmd)
	return ConfigFile(cfg)(cmd)
}

// Test starts a sendxmpp instance and returns a function that runs subtests
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return integration.Test(ctx, cmdName, t, opts...)
}
