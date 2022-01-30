// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net"
	"sync"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type nopContextEncoder struct {
	t xmlstream.Encoder
}

func (n nopContextEncoder) Encode(_ context.Context, v interface{}) error {
	return n.t.Encode(v)
}

type stanzaWriter struct {
	s             *xmpp.Session
	t             xmlstream.Encoder
	sid           string
	acked         bool
	seq           uint16
	to            jid.JID
	writeDeadline time.Time
}

func (w *stanzaWriter) Write(p []byte) (int, error) {
	data := dataPayload{
		Seq:  w.seq,
		SID:  w.sid,
		Data: p,
	}

	ctx := context.Background()
	if !w.writeDeadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, w.writeDeadline)
		defer cancel()
	}

	var e interface {
		Encode(context.Context, interface{}) error
	} = w.s
	if w.t != nil {
		e = nopContextEncoder{w.t}
	}

	var err error
	if w.acked {
		if w.t == nil {
			err = w.s.UnmarshalIQ(ctx, dataIQ{
				IQ: stanza.IQ{
					Type: stanza.SetIQ,
					To:   w.to,
				},
				Data: data,
			}.TokenReader(), nil)
		} else {
			err = e.Encode(ctx, dataIQ{
				IQ: stanza.IQ{
					Type: stanza.SetIQ,
					To:   w.to,
				},
				Data: data,
			})
		}
	} else {
		err = e.Encode(ctx, dataMessage{
			Message: stanza.Message{
				To: w.to,
			},
			Data: data,
		})
	}
	if err != nil {
		return 0, err
	}
	w.seq++
	return len(p), nil
}

// Conn is an IBB stream.
// Writes to the stream are buffered up to the blocksize before being
// transmitted.
type Conn struct {
	closeFlushFunc func() error
	handler        *Handler
	readBuf        *bytes.Buffer
	readLock       sync.Mutex
	readReady      chan struct{}
	writeLock      sync.Mutex
	readDeadline   time.Time
	s              *xmpp.Session
	writeBuf       *bufio.Writer
	seq            uint16
	closed         bool
	stanzaWriter   *stanzaWriter
	maxBufSize     int
}

func newConn(h *Handler, s *xmpp.Session, iq openIQ, recv bool, maxBufSize int) *Conn {
	if maxBufSize < int(iq.Open.BlockSize) && maxBufSize > 0 {
		maxBufSize = 2 * int(iq.Open.BlockSize)
	}
	var to jid.JID
	if recv {
		to = iq.IQ.From
	} else {
		to = iq.IQ.To
	}
	// Setup a buffered writer to write data to the remote entity using stanzas as
	// a carrier.
	stanzaWrite := &stanzaWriter{
		sid:   iq.Open.SID,
		acked: iq.Open.Stanza == "iq" || iq.Open.Stanza == "",
		to:    to,
		s:     s,
	}
	b64Writer := base64.NewEncoder(base64.StdEncoding, stanzaWrite)
	blockSize := iq.Open.BlockSize
	if blockSize == 0 {
		blockSize = BlockSize
	}

	return &Conn{
		readBuf:        bytes.NewBuffer(make([]byte, 0, blockSize)),
		readReady:      make(chan struct{}),
		s:              s,
		writeBuf:       bufio.NewWriterSize(b64Writer, int(blockSize)),
		closeFlushFunc: b64Writer.Close,
		handler:        h,
		stanzaWriter:   stanzaWrite,
		maxBufSize:     maxBufSize,
	}
}

// SID returns a unique session ID for the connection.
func (c *Conn) SID() string {
	return c.stanzaWriter.sid
}

// Stanza returns the carrier stanza type ("message" or "iq") for payloads
// received by the IBB session.
func (c *Conn) Stanza() string {
	if c.stanzaWriter.acked {
		return "iq"
	}
	return "message"
}

// Read reads data from the IBB stream.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	c.readLock.Lock()
	defer c.readLock.Unlock()

	// If the buffer is empty and we would get io.EOF, nil this does not
	// necessarily mean that the connection is closed.
	// In this case wait for a signal that there is more data to read.
	// When the connection is closed this same signal is sent and our final read
	// from the empty buffer will result in 0, io.EOF as expected.
	if c.readBuf.Len() == 0 {
		c.readLock.Unlock()
		<-c.readReady
		c.readLock.Lock()
	}

	return c.readBuf.Read(b)
}

// Write writes data to the IBB stream.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.EOF
	}
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	return c.writeBuf.Write(b)
}

// LocalAddr returns the local network address of the underlying XMPP session.
func (c *Conn) LocalAddr() net.Addr {
	return c.s.LocalAddr()
}

// RemoteAddr returns the remote network address of the IBB stream.
func (c *Conn) RemoteAddr() net.Addr {
	return c.stanzaWriter.to
}

// Size returns the maximum blocksize for data transmitted over the stream.
// Note that individual packets sent on the stream may be less than the block
// size even if there is enough data to fill the block.
func (c *Conn) Size() int {
	return c.writeBuf.Size()
}

// Flush writes any buffered data to the underlying io.Writer.
// This may result in data transfer less than the block size.
func (c *Conn) Flush() error {
	return c.flush(nil)
}

func (c *Conn) flush(t xmlstream.Encoder) error {
	if t == nil {
		c.writeLock.Lock()
		defer c.writeLock.Unlock()
		return c.writeBuf.Flush()
	}

	c.stanzaWriter.t = t
	return c.writeBuf.Flush()
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
// If the write buffer contains data it will be flushed.
func (c *Conn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	// Flush any remaining data to be written.
	err := c.Flush()
	if err != nil {
		return err
	}
	err = c.closeFlushFunc()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if !c.stanzaWriter.writeDeadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, c.stanzaWriter.writeDeadline)
		defer cancel()
	}
	respReadCloser, err := c.s.SendIQElement(ctx, closePayload(c.stanzaWriter.sid), stanza.IQ{
		To:   c.stanzaWriter.to,
		Type: stanza.SetIQ,
	})
	if err != nil {
		return err
	}
	close(c.readReady)
	return respReadCloser.Close()
}

func (c *Conn) closeNoNotify(t xmlstream.Encoder) error {
	if c.closed {
		return nil
	}
	c.closed = true

	c.handler.rmStream(c.stanzaWriter.sid)

	// Flush any remaining data to be written.
	err := c.flush(t)
	if err != nil {
		return err
	}

	close(c.readReady)
	return c.closeFlushFunc()
}

// SetReadBuffer sets the maximum size the internal buffer will be allowed to
// grow to before sending back an error telling the other side to wait before
// transmitting more data.
// The actual buffer is never shrunk, even if a maximum size is set that is less
// than the current length of the buffer.
// Instead, errors will be returned for incoming data until enough data has been
// read from the buffer to shrink it below the new max size.
// If max is zero or less buffer growth is not limited.
// If max is less than the block size it is ignored and the block size is used
// instead.
func (c *Conn) SetReadBuffer(max int) {
	// c.writeBuf.Size() may look out of place here, but it's not about the write
	// buffer itself, it's just the initial block size we negotiated.
	if max < c.writeBuf.Size() && max > 0 {
		max = c.writeBuf.Size()
	}
	c.maxBufSize = max
}

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	c.stanzaWriter.writeDeadline = t
	return nil
}

// SetReadDeadline sets the deadline for future Read calls and any
// currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that some of the
// data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.stanzaWriter.writeDeadline = t
	return nil
}
