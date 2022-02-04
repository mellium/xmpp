// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package aioxmpp facilitates integration testing against aioxmpp.
//
// Tests are written by importing the automatically generated aioxmpp_client
// Python package, subclassing the Daemon class, and overriding its run method.
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
// For more information see aioxmpp_client.py, python/xmpptest.py, and the
// aioxmpp documentation.
package aioxmpp // import "mellium.im/xmpp/internal/integration/aioxmpp"

import (
	"context"
	_ "embed"
	"io"
	"testing"
	"text/template"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/internal/integration/python"
)

const (
	baseFileName = "aioxmpp_client.py"
)

var (
	//go:embed aioxmpp_client.py
	baseTest string
)

func getConfig(cmd *integration.Cmd) python.Config {
	if cmd.Config == nil {
		cmd.Config = python.Config{}
	}
	return cmd.Config.(python.Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	tmpl, err := template.New("aioxmpp").Parse(baseTest)
	if err != nil {
		return err
	}

	return integration.TempFile(baseFileName, func(cmd *integration.Cmd, w io.Writer) error {
		cfg := getConfig(cmd)
		return tmpl.Execute(w, cfg)
	})(cmd)
}

// Test starts the aioxmpp wrapper script and returns a function that runs
// subtests using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return python.Test(ctx, t, opts...)
}
