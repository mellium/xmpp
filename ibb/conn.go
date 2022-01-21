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
		err = e.Encode(ctx, dataIQ{
			IQ: stanza.IQ{
				Type: stanza.SetIQ,
				To:   w.to,
			},
			Data: data,
		})
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
	b64Reader      io.Reader
	readBuf        *bytes.Buffer
	readBufM       *sync.Mutex
	writeLock      sync.Mutex
	readDeadline   time.Time
	s              *xmpp.Session
	writeBuf       *bufio.Writer
	seq            uint16
	closed         bool
	recv           *io.PipeWriter
	stanzaWriter   *stanzaWriter
}

func newConn(h *Handler, s *xmpp.Session, iq openIQ, recv bool) *Conn {
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
	w := bufio.NewWriterSize(b64Writer, int(blockSize))

	// Setup a buffered reader to handle any incoming data that has been decoded
	// by a handler (the handler will write it to the pipe).
	pipeReader, pipeWriter := io.Pipe()
	b64Reader := base64.NewDecoder(base64.StdEncoding, pipeReader)
	r := bytes.NewBuffer(make([]byte, 0, blockSize))
	readBufM := &sync.Mutex{}
	go func() {
		for {
			readBufM.Lock()

			_, err := r.ReadFrom(b64Reader)
			readBufM.Unlock()
			if err != nil {
				return
			}
		}
	}()

	return &Conn{
		b64Reader:      b64Reader,
		readBuf:        r,
		readBufM:       readBufM,
		s:              s,
		writeBuf:       w,
		closeFlushFunc: b64Writer.Close,
		handler:        h,
		recv:           pipeWriter,
		stanzaWriter:   stanzaWrite,
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
	c.readBufM.Lock()
	defer c.readBufM.Unlock()

	// Read first from the buffer, but if that is empty block and read directly
	// from the pipe.
	return io.MultiReader(c.readBuf, c.b64Reader).Read(b)
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

// Size returns the blocksize for the underlying buffer when writing to the IBB
// stream.
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

	respReadCloser, err := c.s.SendIQElement(context.TODO(), closePayload(c.stanzaWriter.sid), stanza.IQ{
		To:   c.stanzaWriter.to,
		Type: stanza.SetIQ,
	})
	if err != nil {
		return err
	}
	err = respReadCloser.Close()
	if err != nil {
		return err
	}

	return c.recv.Close()
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

	err = c.closeFlushFunc()
	if err != nil {
		return err
	}

	return c.recv.Close()
}

// closeError is called when we close the connection due to an error, eg. the
// other side sent invalid base64 and we don't want to continue.
// It removes the connection from tracking without flushing any remaining data
// and without communicating with the server.
func (c *Conn) closeError() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.handler.rmStream(c.stanzaWriter.sid)

	return c.recv.Close()
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
