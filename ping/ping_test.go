// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ping_test

import (
	"mellium.im/xmlstream"
	"mellium.im/xmpp/ping"
)

var (
	_ xmlstream.WriterTo  = ping.IQ{}
	_ xmlstream.Marshaler = ping.IQ{}
)
