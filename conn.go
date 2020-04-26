// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

type tlsConn interface {
	ConnectionState() tls.ConnectionState
}

var _ tlsConn = (*conn)(nil)

// conn is a net.Conn created for the purpose of establishing an XMPP session.
type conn struct {
	c         net.Conn
	rw        io.ReadWriter
	connState func() tls.ConnectionState
}

// newConn wraps an io.ReadWriter in a Conn.
// If rw is already a net.Conn, it is returned without modification.
// If rw is not a conn but prev is, the various Conn methods that are not part
// of io.ReadWriter proxy through to prev.
func newConn(rw io.ReadWriter, prev net.Conn) net.Conn {
	if c, ok := rw.(net.Conn); ok {
		return c
	}

	// Pull out a connection state function if possible.
	tc, ok := rw.(tlsConn)
	if !ok {
		tc, _ = prev.(tlsConn)
	}
	var cs func() tls.ConnectionState
	if tc != nil {
		cs = tc.ConnectionState
	}

	nc := &conn{
		rw:        rw,
		c:         prev,
		connState: cs,
	}
	return nc
}

func (c *conn) ConnectionState() tls.ConnectionState {
	if c.connState == nil {
		return tls.ConnectionState{}
	}
	return c.connState()
}

// Close closes the connection.
func (c *conn) Close() error {
	if c.c != nil {
		return c.c.Close()
	}
	if closer, ok := c.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// LocalAddr returns the local network address.
func (c *conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// Read can be made to time out and return a net.Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *conn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

// RemoteAddr returns the remote network address.
func (c *conn) RemoteAddr() net.Addr {
	if c.c == nil {
		return nil
	}
	return c.c.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *conn) SetDeadline(t time.Time) error {
	if c.c == nil {
		return nil
	}
	return c.c.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *conn) SetReadDeadline(t time.Time) error {
	if c.c == nil {
		return nil
	}
	return c.c.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *conn) SetWriteDeadline(t time.Time) error {
	if c.c == nil {
		return nil
	}
	return c.c.SetWriteDeadline(t)
}

// Write writes data to the connection.
func (c *conn) Write(b []byte) (int, error) {
	return c.rw.Write(b)
}
