// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"golang.org/x/text/language"
)

// Config represents the configuration of an XMPP session.
type Config struct {
	// The default language for any streams constructed using this config.
	Lang language.Tag
}
