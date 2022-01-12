// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package aioxmpp

import (
	_ "embed"
	"io"
	"path/filepath"
	"text/template"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	baseFileName = "aioxmpp_client.py"
	cfgFileName  = "aioxmpp_config.ini"
	cfgFlag      = "-c"
)

var (
	//go:embed aioxmpp_config.ini
	cfgBase string

	//go:embed aioxmpp_client.py
	baseTest string
)

// Config contains options that can be written to the config file.
type Config struct {
	JID      jid.JID
	Password string
	Port     string
	Imports  [][]string
	Args     []string
}

// ConfigFile is an option that can be used to write a temporary config file.
// This will overwrite the existing config file and make most of the other
// options in this package noops.
// This option only exists for the rare occasion that you need complete control
// over the config file.
func ConfigFile(cfg Config) integration.Option {
	cfgTmpl := template.Must(template.New("cfg").Parse(cfgBase))

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
		_ = filepath.Join
		//cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		//return integration.Args(cfgFlag, cfgFilePath)(cmd)
		return nil
	}
}
