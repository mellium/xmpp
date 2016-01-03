// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

type Handler interface {
	Handle(encoder *xml.Encoder, decoder *xml.Decoder) error
}
