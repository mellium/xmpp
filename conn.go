// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"
)

var _ tlsConn = (*teeConn)(nil)
var _ tlsConn = (*conn)(nil)

type tlsConn interface {
	ConnectionState() tls.ConnectionState
}

// conn is a net.Conn created for the purpose of establishing an XMPP session.
type conn struct {
	c         net.Conn
	rw        io.ReadWriter
	rd        func(time.Time) error
	wd        func(time.Time) error
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

	var rd, wd func(time.Time) error
	if rdPrev, ok := prev.(interface {
		SetReadDeadline(time.Time) error
	}); ok {
		rd = rdPrev.SetReadDeadline
	}
	if wdPrev, ok := prev.(interface {
		SetWriteDeadline(time.Time) error
	}); ok {
		wd = wdPrev.SetWriteDeadline
	}

	nc := &conn{
		rw:        rw,
		c:         prev,
		rd:        rd,
		wd:        wd,
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
	if c.rd == nil {
		return nil
	}
	return c.rd(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *conn) SetWriteDeadline(t time.Time) error {
	if c.wd == nil {
		return nil
	}
	return c.wd(t)
}

// Write writes data to the connection.
func (c *conn) Write(b []byte) (int, error) {
	return c.rw.Write(b)
}

// teeConn is a net.Conn that also copies reads and writes to the provided
// writers.
type teeConn struct {
	net.Conn
	tlsConn     *tls.Conn
	ctx         context.Context
	multiWriter io.Writer
	teeReader   io.Reader
}

// newTeeConn creates a teeConn. If the provided context is canceled, writes
// start passing through to the underlying net.Conn and are no longer copied to
// in and out.
func newTeeConn(ctx context.Context, c net.Conn, in, out io.Writer) teeConn {
	if tc, ok := c.(teeConn); ok {
		return tc
	}

	tc := teeConn{Conn: c, ctx: ctx}
	tc.tlsConn, _ = c.(*tls.Conn)
	if in != nil {
		tc.teeReader = io.TeeReader(c, in)
	}
	if out != nil {
		tc.multiWriter = io.MultiWriter(c, out)
	}
	return tc
}

func (tc teeConn) ConnectionState() tls.ConnectionState {
	if tc.tlsConn == nil {
		return tls.ConnectionState{}
	}
	return tc.tlsConn.ConnectionState()
}

func (tc teeConn) Write(p []byte) (int, error) {
	if tc.multiWriter == nil {
		return tc.Conn.Write(p)
	}
	select {
	case <-tc.ctx.Done():
		tc.multiWriter = nil
		return tc.Conn.Write(p)
	default:
	}
	return tc.multiWriter.Write(p)
}

func (tc teeConn) Read(p []byte) (int, error) {
	if tc.teeReader == nil {
		return tc.Conn.Read(p)
	}
	select {
	case <-tc.ctx.Done():
		tc.teeReader = nil
		return tc.Conn.Read(p)
	default:
	}
	return tc.teeReader.Read(p)
}
