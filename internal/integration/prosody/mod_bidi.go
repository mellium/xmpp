// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package prosody

import (
	_ "embed"
	"io"

	"mellium.im/xmpp/internal/integration"
)

//go:embed mod_bidi.lua
var modBidi []byte

// Bidi enables bidirectional S2S connections.
func Bidi() integration.Option {
	// TODO: Once Prosody 0.12 is out this module can be replaced with the builtin
	// mod_s2s_bidi. See https://mellium.im/issue/78
	const modName = "bidi"
	return func(cmd *integration.Cmd) error {
		err := Modules(modName)(cmd)
		if err != nil {
			return err
		}
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := w.Write(modBidi)
			return err
		})(cmd)
	}
}
