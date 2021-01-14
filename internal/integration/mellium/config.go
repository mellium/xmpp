// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mellium

import (
	"mellium.im/xmpp"
)

// Config contains options that can be used to configure the newly started
// server.
type Config struct {
	ListenC2S   bool
	ListenS2S   bool
	C2SFeatures []xmpp.StreamFeature
	S2SFeatures []xmpp.StreamFeature
	LogXML      bool
}
