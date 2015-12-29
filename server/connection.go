// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package server

import "net"

type Handler interface {
	Handle(c net.Conn, l net.Listener) error
}
