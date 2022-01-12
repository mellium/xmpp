// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package prosody

import (
	_ "embed"
	"io"

	"mellium.im/xmpp/internal/integration"
)

//go:embed mod_trustall.lua
var modTrustAll []byte

// TrustAll configures prosody to trust all certificates presented to it without
// any verification.
func TrustAll() integration.Option {
	const modName = "trustall"
	return func(cmd *integration.Cmd) error {
		err := Modules(modName)(cmd)
		if err != nil {
			return err
		}
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := w.Write(modTrustAll)
			return err
		})(cmd)
	}
}
