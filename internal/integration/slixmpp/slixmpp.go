// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package slixmpp facilitates integration testing against slixmpp.
//
// Tests are accomplished by importing the automatically generated
// slixmpp_client Python package, subclassing the Daemon class, and overriding
// the run method.
// Other methods can also be overridden, in particular the prepare_argparse
// method can be used to add command line arguments to the Python scripts.
// For example:
//
//     from slixmpp_client import Daemon
//     import slixmpp
//
//
//     class Ping(Daemon):
//         def prepare_argparse(self) -> None:
//             super().prepare_argparse()
//
//             self.argparse.add_argument(
//                 "-j",
//                 type=slixmpp.jid.JID,
//                 help="The JID to ping",
//             )
//
//         def configure(self):
//             super().configure()
//             self.client.register_plugin('xep_0199') # Ping
//
//         async def run(self) -> None:
//             await self.client.plugin['xep_0199'].ping(jid=self.args.j)
//
// For more information see slixmpp_client.py, python/xmpptest.py, and the
// slixmpp documentation.
package slixmpp // import "mellium.im/xmpp/internal/integration/slixmpp"

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
	baseFileName = "slixmpp_client.py"
)

var (
	//go:embed slixmpp_client.py
	baseTest string
)

func getConfig(cmd *integration.Cmd) python.Config {
	if cmd.Config == nil {
		cmd.Config = python.Config{}
	}
	return cmd.Config.(python.Config)
}

func defaultConfig(cmd *integration.Cmd) error {
	tmpl, err := template.New("slixmpp").Parse(baseTest)
	if err != nil {
		return err
	}

	err = integration.Name("slixmpp")(cmd)
	if err != nil {
		return err
	}

	return integration.TempFile(baseFileName, func(cmd *integration.Cmd, w io.Writer) error {
		cfg := getConfig(cmd)
		return tmpl.Execute(w, cfg)
	})(cmd)
}

// Test starts the slixmpp wrapper script and returns a function that runs
// subtests using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig)
	return python.Test(ctx, t, opts...)
}
