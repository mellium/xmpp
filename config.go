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

	// The authorization identity, and password to authenticate with.
	// Identity is used when a user wants to act on behalf of another user. For
	// instance, an admin might want to log in as another user to help them
	// troubleshoot an issue. Normally it is left blank and the localpart of the
	// Origin JID is used.
	Identity, Password string
}
