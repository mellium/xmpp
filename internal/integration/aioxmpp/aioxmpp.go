// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package aioxmpp facilitates integration testing against aioxmpp.
//
// Tests are accomplished by importing the automatically generated
// aioxmpp_client Python package, subclassing the Daemon class, and overriding
// the run method.
// Other methods can also be overridden, in particular the prepare_argparse
// method can be used to add command line arguments to the Python scripts.
// For example:
//
//     from aioxmpp_client import Daemon
//     import aioxmpp
//
//
//     class Ping(Daemon):
//         def prepare_argparse(self) -> None:
//             super().prepare_argparse()
//
//             def jid(s):
//                 return aioxmpp.JID.fromstr(s)
//             self.argparse.add_argument(
//                 "-j",
//                 type=jid,
//                 help="The JID to ping",
//             )
//
//         async def run(self) -> None:
//             await aioxmpp.ping.ping(self.client, self.args.j)
//
// For more information see aioxmpp_client.py and the aioxmpp documentation.
package aioxmpp // import "mellium.im/xmpp/internal/integration/aioxmpp"

import (
	"context"
	"io"
	"testing"
	"text/template"

	"mellium.im/xmpp/internal/attr"
	"mellium.im/xmpp/internal/integration"
)

const cmdName = "python"

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	tmpl, err := template.New("aioxmpp").Parse(baseTest)
	if err != nil {
		return err
	}

	err = integration.TempFile(baseFileName, func(cmd *integration.Cmd, w io.Writer) error {
		cfg := getConfig(cmd)
		return tmpl.Execute(w, cfg)
	})(cmd)
	if err != nil {
		return err
	}

	cfg := getConfig(cmd)
	return integration.Args(append([]string{baseFileName}, cfg.Args...)...)(cmd)
}

// Import causes the given script to be written out to the working directory and the class name in that script to be
// imported by the main testing runner and executed.
func Import(class, script string) integration.Option {
	return func(cmd *integration.Cmd) error {
		fName := "tmp_" + attr.RandomID()
		cfg := getConfig(cmd)
		cfg.Imports = append(cfg.Imports, []string{fName, class})
		cmd.Config = cfg
		return integration.TempFile(fName+".py", func(cmd *integration.Cmd, w io.Writer) error {
			_, err := io.WriteString(w, script)
			return err
		})(cmd)
	}
}

// Args sets additional command line args to be passed to the script (ie. after the "aioxmpp_client.py" argument).
// If you want to pass arguments to the python process (before the "aioxmpp_client.py" argument) use integration.Args.
func Args(f ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		cfg.Args = append(cfg.Args, f...)
		cmd.Config = cfg
		return nil
	}
}

// Test starts a the aioxmpp wrapper script and returns a function that runs
// subtests using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return integration.Test(ctx, cmdName, t, opts...)
}
