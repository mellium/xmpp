// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"golang.org/x/text/language"
)

// Option's can be used to configure the stream.
type Option func(*options)
type options struct {
	lang          language.Tag
	noVersionAttr bool
	s2sStream     bool
}

func getOpts(o ...Option) (res options) {
	for _, f := range o {
		f(&res)
	}
	return
}

// The Language option specifies the default language for the stream. Clients
// that support multiple languages will assume that all messages, alerts,
// and other textual data in the stream is in the given language unless it is
// specifically overridden.
func Language(l language.Tag) Option {
	return func(o *options) {
		o.lang = l
	}
}

var (
	// The NoVersion option leaves the version attribute off the stream. When the
	// version attribute is missing, servers and clients treat the XMPP version as
	// if it were 0.9. This is an advanced option and generally should not be used
	// except when responding to an incomming stream that has done the same. It
	// does not change the behavior of the stream otherwise (XMPP 1.0 is still
	// used), and may cause problems.
	NoVersion Option = func(o *options) {
		o.noVersionAttr = true
	}
	// The ServerToServer option configures the stream to use the jabber:server
	// namespace.
	ServerToServer = func(o *options) {
		o.s2sStream = true
	}
)
