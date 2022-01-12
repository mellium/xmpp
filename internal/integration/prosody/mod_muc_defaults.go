// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package prosody

import (
	_ "embed"
	"io"

	"mellium.im/xmpp/internal/integration"
)

//go:embed mod_muc_defaults.lua
var modMUCDefaults []byte

// Channel configures the MUC component (if loaded) with a default channel or
// channels.
func Channel(domain string, c ...ChannelConfig) integration.Option {
	const modName = "muc_defaults"
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		comp := cfg.Component[domain]
		comp.MUCDefaults = append(comp.MUCDefaults, c...)
		comp.Modules = append(comp.Modules, modName)
		cfg.Component[domain] = comp
		cmd.Config = cfg
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := w.Write(modMUCDefaults)
			return err
		})(cmd)
	}
}
