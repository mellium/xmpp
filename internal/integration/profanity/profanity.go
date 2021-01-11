// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package profanity facilitates integration testing against Profanity.
package profanity // import "mellium.im/xmpp/internal/integration/profanity"

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	/* #nosec */
	"crypto/sha1"

	"golang.org/x/sys/unix"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName      = "profrc"
	accountsFileName = "accounts"
	tlsFileName      = "tlscerts"
	logFileName      = "profanity.log"
	logFIFOName      = "profanity.log.fifo"
	cmdName          = "profanity"
	profanityFolder  = "profanity"
	configFlag       = "-c"
	logFlag          = "--logfile"
)

// Send transmits the given message or command.
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
	return Send(cmd, fmt.Sprintf("/ping %s", to))
}

// New creates a new, unstarted, Mcabber instance.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error) {
	return integration.New(
		ctx, cmdName,
		opts...,
	)
}

// ConfigFile is an option that can be used to write a temporary config file.
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
		err = integration.TempFile(filepath.Join(profanityFolder, accountsFileName), func(cmd *integration.Cmd, w io.Writer) error {
			return accountsTmpl.Execute(w, struct {
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
		logFIFOPath := filepath.Join(cmd.ConfigDir(), logFIFOName)
		err = unix.Mkfifo(logFIFOPath, 0600)
		if err != nil {
			return err
		}
		return integration.Args(configFlag, cfgFilePath, logFlag, logFIFOPath, "-l", "DEBUG")(cmd)
	}
}

// TrustTLS instructs profanity to trust the provided TLS certificates.
func TrustTLS(cert ...*tls.Certificate) integration.Option {
	return func(cmd *integration.Cmd) error {
		return integration.TempFile(filepath.Join(profanityFolder, tlsFileName), func(cmd *integration.Cmd, w io.Writer) error {
			for _, c := range cert {
				_, err := fmt.Fprintf(w, "[%x]", sha1.Sum(c.Certificate[0]))
				if err != nil {
					return err
				}
			}
			return nil
		})(cmd)
	}
}

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	logFIFOPath := filepath.Join(cmd.ConfigDir(), logFIFOName)
	logFilePath := filepath.Join(cmd.ConfigDir(), logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return err
	}
	err = integration.LogFile(logFIFOPath, logFile)(cmd)
	if err != nil {
		return err
	}

	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	cfg := getConfig(cmd)
	stdin := cmd.Stdin()
	cmd.Config = cfg

	integration.Shutdown(func(cmd *integration.Cmd) error {
		_, err := fmt.Fprint(stdin, "/quit")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdin, "")
		return err
	})(cmd)

	return ConfigFile(cfg)(cmd)
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
