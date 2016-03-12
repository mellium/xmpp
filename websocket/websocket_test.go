// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package websocket

import (
	"net/http"

	"golang.org/x/net/websocket"
)

// Ensure that Handlers are http.Handler's
var _ http.Handler = (Handler)(func(*websocket.Conn) {})
